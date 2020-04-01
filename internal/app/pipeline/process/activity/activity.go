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

package activty

import (
	"context"
	"time"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/process"
)

const ProcessLogActivityName = "process-log"

const ProcessEventActivityName = "process-event"

type Type string
type Status string

const (
	ClusterUpgrade Type = "cluster-upgrade"

	Running  Status = "running"
	Failed   Status = "failed"
	Finished Status = "finished"
)

func NewProcessLogActivity(service process.Service) ProcessLogActivity {
	return ProcessLogActivity{service: service}
}

type ProcessLogActivity struct {
	service process.Service
}

type ProcessLogActivityInput struct {
	ID         string
	ParentID   string
	OrgID      int32
	Name       string
	Type       Type
	ResourceID string
	Status     Status
	StartedAt  time.Time
	FinishedAt *time.Time
}

type ProcessEventActivityInput struct {
	ProcessID string
	Name      string
	Log       string
	Status    Status
	Timestamp time.Time
}

func (a ProcessLogActivity) ExecuteProcessLog(ctx context.Context, input ProcessLogActivityInput) (err error) {
	_, err = a.service.LogProcess(ctx, process.Process{
		Id:         input.ID,
		ParentId:   input.ParentID,
		OrgId:      input.OrgID,
		Name:       input.Name,
		Type:       string(input.Type),
		ResourceId: input.ResourceID,
		Status:     process.ProcessStatus(input.Status),
		StartedAt:  input.StartedAt,
		FinishedAt: input.FinishedAt,
	})

	return
}

func (a ProcessLogActivity) ExecuteProcessEvent(ctx context.Context, input ProcessEventActivityInput) (err error) {
	_, err = a.service.LogProcessEvent(ctx, process.ProcessEvent{
		ProcessId: input.ProcessID,
		Name:      input.Name,
		Log:       input.Log,
		Status:    process.ProcessStatus(input.Status),
		Timestamp: input.Timestamp,
	})

	return
}
