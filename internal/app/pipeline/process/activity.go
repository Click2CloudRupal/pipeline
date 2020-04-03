// Copyright © 2020 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package process

import (
	"context"
	"time"

	"go.uber.org/cadence/workflow"
)

const ProcessLogActivityName = "process-log"

const ProcessEventActivityName = "process-event"

type Status string

const (
	Running  Status = "running"
	Failed   Status = "failed"
	Finished Status = "finished"
)

func NewProcessLogActivity(service Service) ProcessLogActivity {
	return ProcessLogActivity{service: service}
}

type ProcessLogActivity struct {
	service Service
}

type ProcessLogActivityInput struct {
	ID         string
	ParentID   string
	OrgID      int32
	Type       string
	Log        string
	ResourceID string
	Status     Status
	StartedAt  time.Time
	FinishedAt *time.Time
}

type ProcessEventActivityInput struct {
	ProcessID string
	Type      string
	Log       string
	Status    Status
	Timestamp time.Time
}

func (a ProcessLogActivity) ExecuteProcessLog(ctx context.Context, input ProcessLogActivityInput) error {
	_, err := a.service.LogProcess(ctx, Process{
		Id:         input.ID,
		ParentId:   input.ParentID,
		OrgId:      input.OrgID,
		Type:       input.Type,
		Log:        input.Log,
		ResourceId: input.ResourceID,
		Status:     ProcessStatus(input.Status),
		StartedAt:  input.StartedAt,
		FinishedAt: input.FinishedAt,
	})

	return err
}

func (a ProcessLogActivity) ExecuteProcessEvent(ctx context.Context, input ProcessEventActivityInput) error {
	_, err := a.service.LogProcessEvent(ctx, ProcessEvent{
		ProcessId: input.ProcessID,
		Type:      input.Type,
		Log:       input.Log,
		Status:    ProcessStatus(input.Status),
		Timestamp: input.Timestamp,
	})

	return err
}

type ProcessLog struct {
	ctx           workflow.Context
	activityInput ProcessLogActivityInput
}

func (p *ProcessLog) End(err error) {
	finishedAt := workflow.Now(p.ctx)
	p.activityInput.FinishedAt = &finishedAt
	if err != nil {
		p.activityInput.Status = Failed
		p.activityInput.Log = err.Error()
	} else {
		p.activityInput.Status = Finished
	}

	err = workflow.ExecuteActivity(p.ctx, ProcessLogActivityName, p.activityInput).Get(p.ctx, nil)
	if err != nil {
		workflow.GetLogger(p.ctx).Sugar().Warnf("failed to log process end: %s", err)
	}
}

func NewProcessLog(ctx workflow.Context, orgID uint, resourceID string) ProcessLog {
	winfo := workflow.GetInfo(ctx)
	parentID := ""
	if winfo.ParentWorkflowExecution != nil {
		parentID = winfo.ParentWorkflowExecution.ID
	}
	activityInput := ProcessLogActivityInput{
		ID:         winfo.WorkflowExecution.ID,
		ParentID:   parentID,
		Type:       winfo.WorkflowType.Name,
		StartedAt:  workflow.Now(ctx),
		Status:     Running,
		OrgID:      int32(orgID),
		ResourceID: resourceID,
	}
	err := workflow.ExecuteActivity(ctx, ProcessLogActivityName, activityInput).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Sugar().Warnf("failed to log process: %s", err)
	}

	return ProcessLog{ctx: ctx, activityInput: activityInput}
}

type ProcessEventLog struct {
	ctx           workflow.Context
	activityInput ProcessEventActivityInput
}

func (p *ProcessEventLog) End(err error) {
	p.activityInput.Timestamp = workflow.Now(p.ctx)
	if err != nil {
		p.activityInput.Status = Failed
		p.activityInput.Log = err.Error()
	} else {
		p.activityInput.Status = Finished
	}

	err = workflow.ExecuteActivity(p.ctx, ProcessEventActivityName, p.activityInput).Get(p.ctx, nil)
	if err != nil {
		workflow.GetLogger(p.ctx).Sugar().Warnf("failed to log process event end: %s", err.Error())
	}
}

func NewProcessEvent(ctx workflow.Context, activityName string) ProcessEventLog {
	winfo := workflow.GetInfo(ctx)

	activityInput := ProcessEventActivityInput{
		ProcessID: winfo.WorkflowExecution.ID,
		Type:      activityName,
		Timestamp: workflow.Now(ctx),
		Status:    Running,
	}

	err := workflow.ExecuteActivity(ctx, ProcessEventActivityName, activityInput).Get(ctx, nil)
	if err != nil {
		workflow.GetLogger(ctx).Sugar().Warnf("failed to log process event: %s", err)
	}

	return ProcessEventLog{ctx: ctx, activityInput: activityInput}
}
