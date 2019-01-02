package e2e

import (
	"context"
	"testing"
	"time"

	virtuslabv1alpha1 "github.com/VirtusLab/jenkins-operator/pkg/apis/virtuslab/v1alpha1"
	"github.com/VirtusLab/jenkins-operator/pkg/controller/jenkins/configuration/user/seedjobs"
	"github.com/bndr/gojenkins"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestUserConfiguration(t *testing.T) {
	t.Parallel()
	namespace, ctx := setupTest(t)
	// Deletes test namespace
	defer ctx.Cleanup()

	// base
	jenkins := createJenkinsCRWithSeedJob(t, namespace)
	waitForJenkinsBaseConfigurationToComplete(t, jenkins)
	client := verifyJenkinsAPIConnection(t, jenkins)

	// user
	waitForJenkinsUserConfigurationToComplete(t, jenkins)
	verifyJenkinsSeedJobs(t, client, jenkins)
}

func verifyJenkinsSeedJobs(t *testing.T, client *gojenkins.Jenkins, jenkins *virtuslabv1alpha1.Jenkins) {
	// check if job has been configured and executed successfully
	err := wait.Poll(time.Second*10, time.Minute*2, func() (bool, error) {
		t.Logf("Attempting to get configure seed job status '%v'", seedjobs.ConfigureSeedJobsName)
		seedJob, err := client.GetJob(seedjobs.ConfigureSeedJobsName)
		if err != nil || seedJob == nil {
			return false, nil
		}
		build, err := seedJob.GetLastSuccessfulBuild()
		if err != nil || build == nil {
			return false, nil
		}
		return true, nil
	})
	assert.NoError(t, err, "couldn't get jenkins job")

	// WARNING this use case depends on changes in https://github.com/VirtusLab/jenkins-operator-e2e/tree/master/cicd
	seedJobName := "jenkins-operator-e2e-job-dsl-seed" // https://github.com/VirtusLab/jenkins-operator-e2e/blob/master/cicd/jobs/e2e_test_job.jenkins
	err = wait.Poll(time.Second*10, time.Minute*2, func() (bool, error) {
		t.Logf("Attempting to verify if seed job has been created '%v'", seedJobName)
		seedJob, err := client.GetJob(seedJobName)
		if err != nil || seedJob == nil {
			return false, nil
		}
		return true, nil
	})
	assert.NoError(t, err, "couldn't verify if seed job has been created")

	// verify Jenkins.Status.Builds
	// WARNING this use case depends on changes in https://github.com/VirtusLab/jenkins-operator-e2e/tree/master/cicd
	err = framework.Global.Client.Get(context.TODO(), types.NamespacedName{Namespace: jenkins.Namespace, Name: jenkins.Name}, jenkins)
	assert.NoError(t, err, "couldn't get jenkins custom resource")

	assert.NotNil(t, jenkins.Status.Builds)
	assert.NotEmpty(t, jenkins.Status.Builds)
	assert.Equal(t, len(jenkins.Status.Builds), 1)
	build := jenkins.Status.Builds[0]
	assert.Equal(t, build.Name, seedjobs.ConfigureSeedJobsName)
}
