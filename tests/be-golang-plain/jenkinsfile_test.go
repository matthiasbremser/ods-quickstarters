package be_golang_plain

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"

	coreUtils "github.com/opendevstack/ods-core/tests/utils"
	utils "github.com/opendevstack/ods-quickstarters/tests/utils"
)

func TestJenkinsFile(t *testing.T) {

	values, err := utils.ReadConfiguration()
	if err != nil {
		t.Fatal(err)
	}

	_, filename, _, _ := runtime.Caller(0)
	quickstarterPath := filepath.Dir(filename)
	quickstarterName := filepath.Base(quickstarterPath)
	fmt.Printf("quickstarter: %s\n", quickstarterName)
	const componentId = "golang-iq-test"

	// cleanup and create bb resources for this test
	utils.CleanupAndCreateBitbucketProjectAndRepo(
		quickstarterName, componentId)

	// run provision job for quickstarter in project's cd jenkins
	stages, err := utils.RunJenkinsFile(
		"ods-quickstarters",
		values["ODS_BITBUCKET_PROJECT"],
		values["ODS_GIT_REF"],
		coreUtils.PROJECT_NAME,
		fmt.Sprintf("%s/Jenkinsfile", quickstarterName),
		coreUtils.PROJECT_NAME_CD,
		coreUtils.EnvPair{
			Name:  "COMPONENT_ID",
			Value: componentId,
		},
		coreUtils.EnvPair{
			Name:  "GIT_URL_HTTP",
			Value: fmt.Sprintf("%s/%s/%s.git", values["REPO_BASE"], coreUtils.PROJECT_NAME, componentId),
		},
	)

	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Provision Build for %s returned:\n%s", componentId, stages)

	// verify provision jenkins stages - against golden record
	expected, err := ioutil.ReadFile("golden/jenkins-provision-stages.json")
	if err != nil {
		t.Fatal(err)
	}

	expectedAsString := string(expected)
	if stages != expectedAsString {
		t.Fatalf("Actual jenkins stages from prov run: %s don't match -golden:\n'%s'\n-jenkins response:\n'%s'",
			componentId, expectedAsString, stages)
	}

	// run master build of provisioned quickstarter in project's cd jenkins
	stages, err = utils.RunJenkinsFile(
		componentId,
		coreUtils.PROJECT_NAME,
		"master",
		coreUtils.PROJECT_NAME,
		"Jenkinsfile",
		coreUtils.PROJECT_NAME_CD,
		coreUtils.EnvPair{
			Name:  "COMPONENT_ID",
			Value: componentId,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("Master (code) build for %s returned:\n%s", componentId, stages)

	// verify run and build jenkins stages - against golden record
	expected, err = ioutil.ReadFile("golden/jenkins-build-stages.json")
	if err != nil {
		t.Fatal(err)
	}

	expectedAsString = string(expected)
	if stages != expectedAsString {
		t.Fatalf("Actual jenkins stages from build run: %s don't match -golden:\n'%s'\n-jenkins response:\n'%s'",
			componentId, expectedAsString, stages)
	}

	// sonar scan check
	sonarscan, err := utils.RetrieveSonarScan(
		fmt.Sprintf("%s-%s", coreUtils.PROJECT_NAME, componentId))

	if err != nil {
		t.Fatal(err)
	}

	// verify sonar scan - against golden record
	expected, err = ioutil.ReadFile("golden/sonar-scan.json")
	if err != nil {
		t.Fatal(err)
	}

	expectedAsString = string(expected)
	if sonarscan != expectedAsString {
		t.Fatalf("Actual sonar scan for run: %s doesn't match -golden:\n'%s'\n-sonar response:\n'%s'",
			componentId, expectedAsString, sonarscan)
	}

	// get the scrr artifact - this will fail if not found
	stdout, stderr, err = utils.RunScriptFromBaseDir(
		"tests/scripts/get-artifact-from-jenkins-run.sh",
		[]string{
			buildName,
			coreUtils.PROJECT_NAME_CD,
			fmt.Sprintf("SCRR-%s-%s.md", coreUtils.PROJECT_NAME, componentId),
		}, []string{})

	if err != nil {
		t.Fatalf("Could not execute tests/scripts/get-artifact-from-jenkins-run.sh\n - err:%s\nout:%s\nstderr:%s",
			err, stdout, stderr)
	}

	resourcesInTest := coreUtils.Resources{
		Namespace:         coreUtils.PROJECT_NAME_DEV,
		ImageTags:         []coreUtils.ImageTag{{Name: componentId, Tag: "latest"}},
		BuildConfigs:      []string{componentId},
		DeploymentConfigs: []string{componentId},
		Services:          []string{componentId},
		ImageStreams:      []string{componentId},
	}

	coreUtils.CheckResources(resourcesInTest, t)

}
