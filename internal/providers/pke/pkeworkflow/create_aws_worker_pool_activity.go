// Copyright © 2019 Banzai Cloud
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

package pkeworkflow

import (
	"context"
	"fmt"
	"io/ioutil"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"go.uber.org/cadence/activity"
)

const CreateWorkerPoolActivityName = "pke-create-aws-worker-role-activity"

type CreateWorkerPoolActivity struct {
	clusters       Clusters
	tokenGenerator TokenGenerator
}

func NewCreateWorkerPoolActivity(clusters Clusters, tokenGenerator TokenGenerator) *CreateWorkerPoolActivity {
	return &CreateWorkerPoolActivity{
		clusters:       clusters,
		tokenGenerator: tokenGenerator,
	}
}

type CreateWorkerPoolActivityInput struct {
	ClusterID             uint
	Pool                  NodePool
	VPCID                 string
	SubnetID              string
	WorkerInstanceProfile string
	ClusterSecurityGroup  string
	ExternalBaseUrl       string
}

func (a *CreateWorkerPoolActivity) Execute(ctx context.Context, input CreateWorkerPoolActivityInput) (string, error) {
	log := activity.GetLogger(ctx).Sugar().With("clusterID", input.ClusterID)
	cluster, err := a.clusters.GetCluster(ctx, input.ClusterID)
	if err != nil {
		return "", err
	}

	stackName := fmt.Sprintf("pke-pool-%s-worker-%s", cluster.GetName(), input.Pool.Name)

	awsCluster, ok := cluster.(AWSCluster)
	if !ok {
		return "", errors.New(fmt.Sprintf("can't create AWS roles for %t", cluster))
	}

	_, signedToken, err := a.tokenGenerator.GenerateClusterToken(cluster.GetOrganizationId(), cluster.GetID())
	if err != nil {
		return "", emperror.Wrap(err, "can't generate Pipeline token")
	}

	bootstrapCommand, err := awsCluster.GetBootstrapCommand(input.Pool.Name, input.ExternalBaseUrl, signedToken)
	if err != nil {
		return "", emperror.Wrapf(err, "failed to fetch bootstrap command")
	}

	client, err := awsCluster.GetAWSClient()
	if err != nil {
		return "", emperror.Wrap(err, "failed to connect to AWS")
	}

	cfClient := cloudformation.New(client)

	buf, err := ioutil.ReadFile("templates/pke/worker.cf.yaml")
	if err != nil {
		return "", emperror.Wrap(err, "loading CF template")
	}

	clusterName := cluster.GetName()
	stackInput := &cloudformation.CreateStackInput{
		StackName:          aws.String(stackName),
		TemplateBody:       aws.String(string(buf)),
		ClientRequestToken: aws.String(string(activity.GetInfo(ctx).ActivityID)),
		Parameters: []*cloudformation.Parameter{
			{
				ParameterKey:   aws.String("ClusterName"),
				ParameterValue: &clusterName,
			},
			{
				ParameterKey:   aws.String("PkeCommand"),
				ParameterValue: &bootstrapCommand,
			},
			{
				ParameterKey:   aws.String("InstanceType"),
				ParameterValue: aws.String(input.Pool.InstanceType),
			},
			{
				ParameterKey:   aws.String("AvailabilityZones"),
				ParameterValue: aws.String(input.Pool.AvailabilityZones[0]),
			},
			{
				ParameterKey:   aws.String("VPCId"),
				ParameterValue: &input.VPCID,
			},
			{
				ParameterKey:   aws.String("SubnetIds"),
				ParameterValue: &input.SubnetID,
			},
			{
				ParameterKey:   aws.String("IamInstanceProfile"),
				ParameterValue: &input.WorkerInstanceProfile,
			},
			{
				ParameterKey:   aws.String("ImageId"),
				ParameterValue: aws.String("ami-dd3c0f36"),
			},
			{
				ParameterKey:   aws.String("PkeVersion"),
				ParameterValue: aws.String("0.0.5"),
			},
			{
				ParameterKey:   aws.String("KeyName"),
				ParameterValue: aws.String("sanyiMbp"),
			},
			{
				ParameterKey:   aws.String("MinSize"),
				ParameterValue: aws.String(strconv.Itoa(input.Pool.MinCount)),
			},
			{
				ParameterKey:   aws.String("MaxSize"),
				ParameterValue: aws.String(strconv.Itoa(input.Pool.MaxCount)),
			},
			{
				ParameterKey:   aws.String("DesiredCapacity"),
				ParameterValue: aws.String(strconv.Itoa(input.Pool.Count)),
			},
			{
				ParameterKey:   aws.String("ClusterSecurityGroup"),
				ParameterValue: aws.String(input.ClusterSecurityGroup),
			},
		},
	}

	output, err := cfClient.CreateStack(stackInput)
	if err, ok := err.(awserr.Error); ok {
		switch err.Code() {
		case cloudformation.ErrCodeAlreadyExistsException:
			log.Infof("stack already exists: %s", err.Message())
		default:
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	return *output.StackId, nil
}