package goldpushk

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.skia.org/infra/go/exec"
	"go.skia.org/infra/go/git"
	"go.skia.org/infra/go/git/testutils"
	"go.skia.org/infra/go/testutils/unittest"
)

func TestNew(t *testing.T) {
	unittest.SmallTest(t)

	// Gather some DeployableUnits to pass to New() as parameters.
	s := ProductionDeployableUnits()
	deployableUnits := []DeployableUnit{}
	deployableUnits = appendUnit(t, deployableUnits, s, Skia, DiffServer)            // Regular deployment.
	deployableUnits = appendUnit(t, deployableUnits, s, SkiaPublic, SkiaCorrectness) // Public deployment with non-templated ConfigMap.
	canariedDeployableUnits := []DeployableUnit{}
	canariedDeployableUnits = appendUnit(t, canariedDeployableUnits, s, Skia, IngestionBT)    // Regular deployment with templated ConfigMap.
	canariedDeployableUnits = appendUnit(t, canariedDeployableUnits, s, Fuchsia, DiffServer)  // Internal deployment.
	canariedDeployableUnits = appendUnit(t, canariedDeployableUnits, s, Fuchsia, IngestionBT) // Internal deployment with templated ConfigMap.

	// Call code under test.
	g := New(deployableUnits, canariedDeployableUnits, "path/to/buildbot", true, true, "http://skia-public.com", "http://skia-corp.com")

	expected := &Goldpushk{
		deployableUnits:         deployableUnits,
		canariedDeployableUnits: canariedDeployableUnits,
		rootPath:                "path/to/buildbot",
		dryRun:                  true,
		noCommit:                true,
		skiaPublicConfigRepoUrl: "http://skia-public.com",
		skiaCorpConfigRepoUrl:   "http://skia-corp.com",
	}
	assert.Equal(t, expected, g)
}

// TODO(lovisolo): Implement and test.
func TestGoldpushkRun(t *testing.T) {
	unittest.SmallTest(t)

	t.Skip("Not implemented")
}

func TestGoldpushkCheckOutGitRepositories(t *testing.T) {
	unittest.MediumTest(t)

	ctx := context.Background()

	// Create two fake skia-{public,corp}-config repositories (i.e. "git init" two temp directories).
	fakeSkiaPublicConfig, fakeSkiaCorpConfig := createFakeConfigRepos(t, ctx)
	defer fakeSkiaPublicConfig.Cleanup()
	defer fakeSkiaCorpConfig.Cleanup()

	// Create the goldpushk instance under test. We pass it the file://... URLs to
	// the two Git repositories created earlier.
	g := Goldpushk{
		skiaPublicConfigRepoUrl: fakeSkiaPublicConfig.RepoUrl(),
		skiaCorpConfigRepoUrl:   fakeSkiaCorpConfig.RepoUrl(),
	}

	// Hide goldpushk output to stdout.
	restoreStdout := hideStdout(t)
	defer restoreStdout()

	// Check out the fake "skia-public-config" and "skia-corp-config"
	// repositories. This will clone the repositories created earlier by running
	// "git clone file://...".
	err := g.checkOutGitRepositories(ctx)

	// Assert that no errors occurred and that we have a git.TempCheckout instance
	// for each cloned repo.
	assert.NoError(t, err)
	assert.NotNil(t, g.skiaPublicConfigCheckout)
	assert.NotNil(t, g.skiaCorpConfigCheckout)

	// Clean up the checkouts after the test finishes.
	defer g.skiaPublicConfigCheckout.Delete()
	defer g.skiaCorpConfigCheckout.Delete()

	// Assert that the local path to the checkouts is not the same as the local
	// path to the fake "skia-public-config" and "skia-corp-config" repos created
	// earlier. This is just a basic sanity check that ensures that we're actually
	// dealing with clones of the original repos, as opposed to the original repos
	// themselves.
	assert.NotEqual(t, g.skiaPublicConfigCheckout.GitDir, fakeSkiaPublicConfig.Dir())
	assert.NotEqual(t, g.skiaCorpConfigCheckout.GitDir, fakeSkiaCorpConfig.Dir())

	// Read files from the checkouts.
	publicWhichRepoTxtBytes, err := ioutil.ReadFile(filepath.Join(string(g.skiaPublicConfigCheckout.GitDir), "which-repo.txt"))
	assert.NoError(t, err)
	corpWhichRepoTxtBytes, err := ioutil.ReadFile(filepath.Join(string(g.skiaCorpConfigCheckout.GitDir), "which-repo.txt"))
	assert.NoError(t, err)

	// Assert that the contents of file "which-repo.txt" on each checkout matches
	// the contents of the same file on the corresponding origin repository.
	assert.Equal(t, "This is repo skia-public-config!", string(publicWhichRepoTxtBytes))
	assert.Equal(t, "This is repo skia-corp-config!", string(corpWhichRepoTxtBytes))
}

func TestGoldpushkGetDeploymentFilePath(t *testing.T) {
	unittest.SmallTest(t)

	// Create the goldpushk instance under test.
	g := Goldpushk{}
	addFakeConfigRepoCheckouts(&g)

	// Gather the DeployableUnits we will call Goldpushk.getDeploymentFilePath() with.
	s := ProductionDeployableUnits()
	publicUnit, _ := s.Get(makeID(Skia, DiffServer))
	internalUnit, _ := s.Get(makeID(Fuchsia, DiffServer))

	assert.Equal(t, filepath.Join(g.skiaPublicConfigCheckout.Dir(), "gold-skia-diffserver.yaml"), g.getDeploymentFilePath(publicUnit))
	assert.Equal(t, filepath.Join(g.skiaCorpConfigCheckout.Dir(), "gold-fuchsia-diffserver.yaml"), g.getDeploymentFilePath(internalUnit))
}

func TestGoldpushkGetConfigMapFilePath(t *testing.T) {
	unittest.SmallTest(t)

	// Create the goldpushk instance under test.
	skiaInfraRoot := "/path/to/buildbot"
	g := Goldpushk{
		rootPath: skiaInfraRoot,
	}
	addFakeConfigRepoCheckouts(&g)

	// Gather the DeployableUnits we will call Goldpushk.getConfigMapFilePath() with.
	s := ProductionDeployableUnits()
	publicUnitWithoutConfigMap, _ := s.Get(makeID(Skia, DiffServer))
	publicUnitWithConfigMapTemplate, _ := s.Get(makeID(Skia, IngestionBT))
	publicUnitWithConfigMapFile, _ := s.Get(makeID(SkiaPublic, SkiaCorrectness))
	internalUnitWithoutConfigMap, _ := s.Get(makeID(Fuchsia, DiffServer))
	internalUnitWithConfigMapTemplate, _ := s.Get(makeID(Fuchsia, IngestionBT))

	// Helper functions to write more concise assertions.
	assertNoConfigMap := func(unit DeployableUnit) {
		_, ok := g.getConfigMapFilePath(unit)
		assert.False(t, ok, unit.CanonicalName())
	}
	assertConfigMapFileEquals := func(unit DeployableUnit, expectedPath ...string) {
		path, ok := g.getConfigMapFilePath(unit)
		assert.True(t, ok, unit.CanonicalName())
		assert.Equal(t, filepath.Join(expectedPath...), path, unit.CanonicalName())
	}

	// Get the paths to the checked out repositories.
	skiaPublicConfigPath := g.skiaPublicConfigCheckout.Dir()
	skiaCorpConfigPath := g.skiaCorpConfigCheckout.Dir()

	// Assert that we get the correct ConfigMap file path for each DeployableUnit.
	assertNoConfigMap(publicUnitWithoutConfigMap)
	assertConfigMapFileEquals(publicUnitWithConfigMapTemplate, skiaPublicConfigPath, "gold-skia-ingestion-config-bt.json5")
	assertConfigMapFileEquals(publicUnitWithConfigMapFile, skiaInfraRoot, "golden/k8s-instances/skia-public/authorized-params.json5")
	assertNoConfigMap(internalUnitWithoutConfigMap)
	assertConfigMapFileEquals(internalUnitWithConfigMapTemplate, skiaCorpConfigPath, "gold-fuchsia-ingestion-config-bt.json5")
}

func TestRegenerateConfigFiles(t *testing.T) {
	unittest.SmallTest(t)

	// Test on a good combination of different types of deployments.
	s := ProductionDeployableUnits()
	deployableUnits := []DeployableUnit{}
	deployableUnits = appendUnit(t, deployableUnits, s, Skia, DiffServer)            // Regular deployment.
	deployableUnits = appendUnit(t, deployableUnits, s, SkiaPublic, SkiaCorrectness) // Public deployment with non-templated ConfigMap.
	canariedDeployableUnits := []DeployableUnit{}
	canariedDeployableUnits = appendUnit(t, canariedDeployableUnits, s, Skia, IngestionBT)    // Regular deployment with templated ConfigMap.
	canariedDeployableUnits = appendUnit(t, canariedDeployableUnits, s, Fuchsia, DiffServer)  // Internal deployment.
	canariedDeployableUnits = appendUnit(t, canariedDeployableUnits, s, Fuchsia, IngestionBT) // Internal deployment with templated ConfigMap.

	// Create the goldpushk instance under test.
	g := Goldpushk{
		deployableUnits:         deployableUnits,
		canariedDeployableUnits: canariedDeployableUnits,
		rootPath:                "/path/to/buildbot",
	}
	addFakeConfigRepoCheckouts(&g)

	// Get the paths to the checked out repositories, ending with a separator.
	skiaPublicConfigPath := g.skiaPublicConfigCheckout.Dir() + string(filepath.Separator)
	skiaCorpConfigPath := g.skiaCorpConfigCheckout.Dir() + string(filepath.Separator)

	// Set up mocks.
	commandCollector := exec.CommandCollector{}
	commandCollectorCtx := exec.NewContext(context.Background(), commandCollector.Run)

	// Call code under test.
	err := g.regenerateConfigFiles(commandCollectorCtx)
	assert.NoError(t, err)

	// Expected commands.
	expected := []string{
		// Skia DiffServer
		"kube-conf-gen " +
			"-c /path/to/buildbot/golden/k8s-config-templates/gold-common.json5 " +
			"-c /path/to/buildbot/golden/k8s-instances/skia-instance.json5 " +
			"-extra INSTANCE_ID:skia " +
			"-t /path/to/buildbot/golden/k8s-config-templates/gold-diffserver-template.yaml " +
			"-parse_conf=false " +
			"-strict " +
			"-o " + skiaPublicConfigPath + "gold-skia-diffserver.yaml",

		// SkiaPublic SkiaCorrectness
		"kube-conf-gen " +
			"-c /path/to/buildbot/golden/k8s-config-templates/gold-common.json5 " +
			"-c /path/to/buildbot/golden/k8s-instances/skia-public-instance.json5 " +
			"-extra INSTANCE_ID:skia-public " +
			"-t /path/to/buildbot/golden/k8s-config-templates/gold-skiacorrectness-template.yaml " +
			"-parse_conf=false " +
			"-strict " +
			"-o " + skiaPublicConfigPath + "gold-skia-public-skiacorrectness.yaml",

		// Skia IngestionBT
		"kube-conf-gen " +
			"-c /path/to/buildbot/golden/k8s-config-templates/gold-common.json5 " +
			"-c /path/to/buildbot/golden/k8s-instances/skia-instance.json5 " +
			"-extra INSTANCE_ID:skia " +
			"-t /path/to/buildbot/golden/k8s-config-templates/gold-ingestion-bt-template.yaml " +
			"-parse_conf=false " +
			"-strict " +
			"-o " + skiaPublicConfigPath + "gold-skia-ingestion-bt.yaml",

		// Skia IngestionBT ConfigMap
		"kube-conf-gen " +
			"-c /path/to/buildbot/golden/k8s-config-templates/gold-common.json5 " +
			"-c /path/to/buildbot/golden/k8s-instances/skia-instance.json5 " +
			"-extra INSTANCE_ID:skia " +
			"-t /path/to/buildbot/golden/k8s-config-templates/ingest-config-template.json5 " +
			"-parse_conf=false " +
			"-strict " +
			"-o " + skiaPublicConfigPath + "gold-skia-ingestion-config-bt.json5",

		// Fuchsia DiffServer
		"kube-conf-gen " +
			"-c /path/to/buildbot/golden/k8s-config-templates/gold-common.json5 " +
			"-c /path/to/buildbot/golden/k8s-instances/fuchsia-instance.json5 " +
			"-extra INSTANCE_ID:fuchsia " +
			"-t /path/to/buildbot/golden/k8s-config-templates/gold-diffserver-template.yaml " +
			"-parse_conf=false " +
			"-strict " +
			"-o " + skiaCorpConfigPath + "gold-fuchsia-diffserver.yaml",

		// Fuchsia IngestionBT
		"kube-conf-gen " +
			"-c /path/to/buildbot/golden/k8s-config-templates/gold-common.json5 " +
			"-c /path/to/buildbot/golden/k8s-instances/fuchsia-instance.json5 " +
			"-extra INSTANCE_ID:fuchsia " +
			"-t /path/to/buildbot/golden/k8s-config-templates/gold-ingestion-bt-template.yaml " +
			"-parse_conf=false " +
			"-strict " +
			"-o " + skiaCorpConfigPath + "gold-fuchsia-ingestion-bt.yaml",

		// Fuchsia IngestionBT ConfigMap
		"kube-conf-gen " +
			"-c /path/to/buildbot/golden/k8s-config-templates/gold-common.json5 " +
			"-c /path/to/buildbot/golden/k8s-instances/fuchsia-instance.json5 " +
			"-extra INSTANCE_ID:fuchsia " +
			"-t /path/to/buildbot/golden/k8s-config-templates/ingest-config-template.json5 " +
			"-parse_conf=false " +
			"-strict " +
			"-o " + skiaCorpConfigPath + "gold-fuchsia-ingestion-config-bt.json5",
	}

	for i, e := range expected {
		assert.Equal(t, e, exec.DebugString(commandCollector.Commands()[i]))
	}
}

func TestCommitConfigFiles(t *testing.T) {
	unittest.MediumTest(t)

	ctx := context.Background()

	// Create two fake skia-{public,corp}-config repositories (i.e. "git init" two temp directories).
	fakeSkiaPublicConfig, fakeSkiaCorpConfig := createFakeConfigRepos(t, ctx)
	defer fakeSkiaPublicConfig.Cleanup()
	defer fakeSkiaCorpConfig.Cleanup()

	// Assert that there is just one commit on both repositories.
	assertNumCommits(t, ctx, fakeSkiaPublicConfig, 1)
	assertNumCommits(t, ctx, fakeSkiaCorpConfig, 1)

	// Create the goldpushk instance under test. We pass it the file://... URLs to the two Git
	// repositories created earlier.
	g := Goldpushk{
		skiaPublicConfigRepoUrl: fakeSkiaPublicConfig.RepoUrl(),
		skiaCorpConfigRepoUrl:   fakeSkiaCorpConfig.RepoUrl(),
	}

	// Hide goldpushk output to stdout.
	restoreStdout := hideStdout(t)
	defer restoreStdout()

	// Check out the fake "skia-public-config" and "skia-corp-config" repositories created earlier.
	// This will run "git clone file://..." for each repository.
	err := g.checkOutGitRepositories(ctx)
	assert.NoError(t, err)
	defer g.skiaPublicConfigCheckout.Delete()
	defer g.skiaCorpConfigCheckout.Delete()

	// Add changes to skia-public-config.
	writeFileIntoRepo(t, g.skiaPublicConfigCheckout, "foo.yaml", "I'm a change in skia-public-config.")
	writeFileIntoRepo(t, g.skiaCorpConfigCheckout, "bar.yaml", "I'm a change in skia-corp-config.")

	// Pretend that the user confirms the commit step.
	cleanup := fakeStdin(t, "y\n")
	defer cleanup()

	// Call the function under test, which will try to commit and push the changes.
	ok, err := g.commitConfigFiles(ctx)
	assert.NoError(t, err)

	// Assert that the user confirmed the commit step.
	assert.True(t, ok)

	// Assert that the changes were pushed to the fake skia-{public,corp}-config repositories.
	assertNumCommits(t, ctx, fakeSkiaPublicConfig, 2)
	assertNumCommits(t, ctx, fakeSkiaCorpConfig, 2)
	assertRepositoryContainsFileWithContents(t, ctx, fakeSkiaPublicConfig, "foo.yaml", "I'm a change in skia-public-config.")
	assertRepositoryContainsFileWithContents(t, ctx, fakeSkiaCorpConfig, "bar.yaml", "I'm a change in skia-corp-config.")
}

func TestCommitConfigFilesAbortedByUser(t *testing.T) {
	unittest.MediumTest(t)

	ctx := context.Background()

	// Create two fake skia-{public,corp}-config repositories (i.e. "git init" two temp directories).
	fakeSkiaPublicConfig, fakeSkiaCorpConfig := createFakeConfigRepos(t, ctx)
	defer fakeSkiaPublicConfig.Cleanup()
	defer fakeSkiaCorpConfig.Cleanup()

	// Assert that there is just one commit on both repositories.
	assertNumCommits(t, ctx, fakeSkiaPublicConfig, 1)
	assertNumCommits(t, ctx, fakeSkiaCorpConfig, 1)

	// Create the goldpushk instance under test. We pass it the file://... URLs to the two Git
	// repositories created earlier.
	g := Goldpushk{
		skiaPublicConfigRepoUrl: fakeSkiaPublicConfig.RepoUrl(),
		skiaCorpConfigRepoUrl:   fakeSkiaCorpConfig.RepoUrl(),
	}

	// Hide goldpushk output to stdout.
	restoreStdout := hideStdout(t)
	defer restoreStdout()

	// Check out the fake "skia-public-config" and "skia-corp-config" repositories created earlier.
	// This will run "git clone file://..." for each repository.
	err := g.checkOutGitRepositories(ctx)
	assert.NoError(t, err)
	defer g.skiaPublicConfigCheckout.Delete()
	defer g.skiaCorpConfigCheckout.Delete()

	// Add changes to skia-public-config and skia-corp-config.
	writeFileIntoRepo(t, g.skiaPublicConfigCheckout, "foo.yaml", "I'm a change in skia-public-config.")
	writeFileIntoRepo(t, g.skiaCorpConfigCheckout, "bar.yaml", "I'm a change in skia-corp-config.")

	// Pretend that the user aborts the commit step.
	restoreStdin := fakeStdin(t, "n\n")
	defer restoreStdin()

	// Call the function under test, which will try to commit and push the changes.
	ok, err := g.commitConfigFiles(ctx)
	assert.NoError(t, err)

	// Assert that the user aborted the commit step.
	assert.False(t, ok)

	// Assert that no changes were pushed to skia-public-config or skia-corp-config.
	assertNumCommits(t, ctx, fakeSkiaPublicConfig, 1)
	assertNumCommits(t, ctx, fakeSkiaCorpConfig, 1)
}

func TestCommitConfigFilesSkipped(t *testing.T) {
	unittest.MediumTest(t)

	ctx := context.Background()

	// Create two fake skia-{public,corp}-config repositories (i.e. "git init" two temp directories).
	fakeSkiaPublicConfig, fakeSkiaCorpConfig := createFakeConfigRepos(t, ctx)
	defer fakeSkiaPublicConfig.Cleanup()
	defer fakeSkiaCorpConfig.Cleanup()

	// Assert that there is just one commit on both repositories.
	assertNumCommits(t, ctx, fakeSkiaPublicConfig, 1)
	assertNumCommits(t, ctx, fakeSkiaCorpConfig, 1)

	// Create the goldpushk instance under test. We pass it the file://... URLs to the two Git
	// repositories created earlier.
	g := Goldpushk{
		skiaPublicConfigRepoUrl: fakeSkiaPublicConfig.RepoUrl(),
		skiaCorpConfigRepoUrl:   fakeSkiaCorpConfig.RepoUrl(),
		noCommit:                true,
	}

	// Hide goldpushk output to stdout.
	restoreStdout := hideStdout(t)
	defer restoreStdout()

	// Check out the fake "skia-public-config" and "skia-corp-config" repositories created earlier.
	// This will run "git clone file://..." for each repository.
	err := g.checkOutGitRepositories(ctx)
	assert.NoError(t, err)
	defer g.skiaPublicConfigCheckout.Delete()
	defer g.skiaCorpConfigCheckout.Delete()

	// Add changes to skia-public-config and skia-corp-config.
	writeFileIntoRepo(t, g.skiaPublicConfigCheckout, "foo.yaml", "I'm a change in skia-public-config.")
	writeFileIntoRepo(t, g.skiaCorpConfigCheckout, "bar.yaml", "I'm a change in skia-corp-config.")

	// Call the function under test, which should not commit nor push any changes.
	ok, err := g.commitConfigFiles(ctx)
	assert.NoError(t, err)
	assert.True(t, ok)

	// Assert that no changes were pushed to skia-public-config or skia-corp-config.
	assertNumCommits(t, ctx, fakeSkiaPublicConfig, 1)
	assertNumCommits(t, ctx, fakeSkiaCorpConfig, 1)
}

func TestSwitchClusters(t *testing.T) {
	unittest.SmallTest(t)

	// Create the goldpushk instance under test.
	g := Goldpushk{}

	// Set up mocks.
	commandCollector := exec.CommandCollector{}
	commandCollectorCtx := exec.NewContext(context.Background(), commandCollector.Run)

	// Test cases.
	testCases := []struct {
		cluster     cluster
		expectedCmd string
	}{
		{
			cluster:     clusterSkiaPublic,
			expectedCmd: "gcloud container clusters get-credentials skia-public --zone us-central1-a --project skia-public",
		},
		{
			cluster:     clusterSkiaCorp,
			expectedCmd: "gcloud container clusters get-credentials skia-corp --zone us-central1-a --project google.com:skia-corp",
		},
	}

	for i, tc := range testCases {
		err := g.switchClusters(commandCollectorCtx, tc.cluster)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedCmd, exec.DebugString(commandCollector.Commands()[i]))
	}
}

func TestPushSingleDeployableUnitDeleteNonexistentConfigMap(t *testing.T) {
	unittest.SmallTest(t)

	// Gather the DeployableUnit to deploy.
	s := ProductionDeployableUnits()
	unit, ok := s.Get(makeID(Skia, IngestionBT))
	assert.True(t, ok)

	// Create the goldpushk instance under test.
	g := &Goldpushk{}
	addFakeConfigRepoCheckouts(g)

	// Set up mocks.
	commandCollector := exec.CommandCollector{}
	commandCollector.SetDelegateRun(func(ctx context.Context, cmd *exec.Command) error {
		if cmd.Name == "kubectl" && cmd.Args[0] == "delete" {
			// This is the actual error message that is returned when the command exits with status 1.
			return errors.New("Command exited with exit status 1: kubectl delete configmap gold-skia-ingestion-config-bt")
		}
		return nil
	})
	commandCollectorCtx := exec.NewContext(context.Background(), commandCollector.Run)

	// Call code under test.
	err := g.pushSingleDeployableUnit(commandCollectorCtx, unit)
	assert.NoError(t, err)

	// Assert that the correct kubectl and gcloud commands were executed.
	expectedCommands := []string{
		"gcloud container clusters get-credentials skia-public --zone us-central1-a --project skia-public",
		"kubectl delete configmap gold-skia-ingestion-config-bt",
		"kubectl create configmap gold-skia-ingestion-config-bt --from-file /path/to/skia-public-config/gold-skia-ingestion-config-bt.json5",
		"kubectl apply -f /path/to/skia-public-config/gold-skia-ingestion-bt.yaml",
	}
	assert.Len(t, commandCollector.Commands(), len(expectedCommands))
	for i, command := range expectedCommands {
		assert.Equal(t, command, exec.DebugString(commandCollector.Commands()[i]))
	}
}

func TestPushCanaries(t *testing.T) {
	unittest.SmallTest(t)

	// Gather the DeployableUnits to deploy.
	s := ProductionDeployableUnits()
	units := []DeployableUnit{}
	units = appendUnit(t, units, s, Skia, DiffServer)     // Public.
	units = appendUnit(t, units, s, Skia, IngestionBT)    // Public, with config map.
	units = appendUnit(t, units, s, Fuchsia, DiffServer)  // Internal.
	units = appendUnit(t, units, s, Fuchsia, IngestionBT) // Internal, with config map.

	// Create the goldpushk instance under test.
	g := &Goldpushk{
		canariedDeployableUnits: units,
	}
	addFakeConfigRepoCheckouts(g)

	// Hide goldpushk output to stdout.
	restoreStdout := hideStdout(t)
	defer restoreStdout()

	// Set up mocks.
	commandCollector := exec.CommandCollector{}
	commandCollectorCtx := exec.NewContext(context.Background(), commandCollector.Run)

	// Call code under test.
	err := g.pushCanaries(commandCollectorCtx)
	assert.NoError(t, err)

	// Assert that the correct kubectl and gcloud commands were executed.
	expectedCommands := []string{
		"gcloud container clusters get-credentials skia-public --zone us-central1-a --project skia-public",
		"kubectl apply -f /path/to/skia-public-config/gold-skia-diffserver.yaml",
		"kubectl delete configmap gold-skia-ingestion-config-bt",
		"kubectl create configmap gold-skia-ingestion-config-bt --from-file /path/to/skia-public-config/gold-skia-ingestion-config-bt.json5",
		"kubectl apply -f /path/to/skia-public-config/gold-skia-ingestion-bt.yaml",
		"gcloud container clusters get-credentials skia-corp --zone us-central1-a --project google.com:skia-corp",
		"kubectl apply -f /path/to/skia-corp-config/gold-fuchsia-diffserver.yaml",
		"kubectl delete configmap gold-fuchsia-ingestion-config-bt",
		"kubectl create configmap gold-fuchsia-ingestion-config-bt --from-file /path/to/skia-corp-config/gold-fuchsia-ingestion-config-bt.json5",
		"kubectl apply -f /path/to/skia-corp-config/gold-fuchsia-ingestion-bt.yaml",
	}
	assert.Len(t, commandCollector.Commands(), len(expectedCommands))
	for i, command := range expectedCommands {
		assert.Equal(t, command, exec.DebugString(commandCollector.Commands()[i]))
	}
}

func TestPushServices(t *testing.T) {
	unittest.SmallTest(t)

	// Gather the DeployableUnits to deploy.
	s := ProductionDeployableUnits()
	units := []DeployableUnit{}
	units = appendUnit(t, units, s, Skia, DiffServer)     // Public.
	units = appendUnit(t, units, s, Skia, IngestionBT)    // Public, with config map.
	units = appendUnit(t, units, s, Fuchsia, DiffServer)  // Internal.
	units = appendUnit(t, units, s, Fuchsia, IngestionBT) // Internal, with config map.

	// Create the goldpushk instance under test.
	g := &Goldpushk{
		deployableUnits: units,
	}
	addFakeConfigRepoCheckouts(g)

	// Hide goldpushk output to stdout.
	restoreStdout := hideStdout(t)
	defer restoreStdout()

	// Set up mocks.
	commandCollector := exec.CommandCollector{}
	commandCollectorCtx := exec.NewContext(context.Background(), commandCollector.Run)

	// Call code under test.
	err := g.pushServices(commandCollectorCtx)
	assert.NoError(t, err)

	// Assert that the correct kubectl and gcloud commands were executed.
	expectedCommands := []string{
		"gcloud container clusters get-credentials skia-public --zone us-central1-a --project skia-public",
		"kubectl apply -f /path/to/skia-public-config/gold-skia-diffserver.yaml",
		"kubectl delete configmap gold-skia-ingestion-config-bt",
		"kubectl create configmap gold-skia-ingestion-config-bt --from-file /path/to/skia-public-config/gold-skia-ingestion-config-bt.json5",
		"kubectl apply -f /path/to/skia-public-config/gold-skia-ingestion-bt.yaml",
		"gcloud container clusters get-credentials skia-corp --zone us-central1-a --project google.com:skia-corp",
		"kubectl apply -f /path/to/skia-corp-config/gold-fuchsia-diffserver.yaml",
		"kubectl delete configmap gold-fuchsia-ingestion-config-bt",
		"kubectl create configmap gold-fuchsia-ingestion-config-bt --from-file /path/to/skia-corp-config/gold-fuchsia-ingestion-config-bt.json5",
		"kubectl apply -f /path/to/skia-corp-config/gold-fuchsia-ingestion-bt.yaml",
	}
	assert.Len(t, commandCollector.Commands(), len(expectedCommands))
	for i, command := range expectedCommands {
		assert.Equal(t, command, exec.DebugString(commandCollector.Commands()[i]))
	}
}

func TestGetUptimesSingleCluster(t *testing.T) {
	unittest.SmallTest(t)

	// Gather the DeployableUnits to deploy.
	s := ProductionDeployableUnits()
	units := []DeployableUnit{}
	units = appendUnit(t, units, s, Chrome, BaselineServer)
	units = appendUnit(t, units, s, ChromeGPU, BaselineServer)
	units = appendUnit(t, units, s, Flutter, BaselineServer)
	units = appendUnit(t, units, s, Flutter, DiffServer)

	// Create the goldpushk instance under test.
	g := &Goldpushk{}

	// Set up mocks.
	commandCollector := exec.CommandCollector{}
	commandCollector.SetDelegateRun(func(ctx context.Context, cmd *exec.Command) error {
		n, err := cmd.CombinedOutput.Write([]byte(kubectlGetPodsOutput))
		assert.NoError(t, err)
		assert.Equal(t, len(kubectlGetPodsOutput), n)
		return nil
	})
	commandCollectorCtx := exec.NewContext(context.Background(), commandCollector.Run)

	// Fake time.
	now := time.Date(2019, 9, 24, 17, 58, 2, 0, time.UTC) // 2019-09-24T17:58:02Z

	// Call code under test.
	uptime, err := g.getUptimesSingleCluster(commandCollectorCtx, units, now)
	assert.NoError(t, err)

	// Assert that we get the expected uptimes.
	assert.Len(t, uptime, 2)
	assert.Equal(t, 29*time.Second, uptime[makeID(Chrome, BaselineServer)])     // 17:58:02 - 17:57:33
	assert.Equal(t, 159*time.Second, uptime[makeID(ChromeGPU, BaselineServer)]) // 17:58:02 - 17:55:23

	// One of its containers is not running (see line "gold-flutter-baselineserver ... <none>" above).
	assert.NotContains(t, uptime, makeID(Flutter, BaselineServer))

	// Its only container is not running (see line "gold-flutter-diffserver ... <none>" above).
	assert.NotContains(t, uptime, makeID(Flutter, DiffServer))
}

// appendUnit will retrieve a DeployableUnit from the given DeployableUnitSet using the given
// Instance and Service and append it to the given DeployableUnit slice.
func appendUnit(t *testing.T, units []DeployableUnit, s DeployableUnitSet, instance Instance, service Service) []DeployableUnit {
	unit, ok := s.Get(DeployableUnitID{Instance: instance, Service: service})
	assert.True(t, ok)
	return append(units, unit)
}

// makeID is a convenience method to create a DeployableUnitID.
func makeID(instance Instance, service Service) DeployableUnitID {
	return DeployableUnitID{
		Instance: instance,
		Service:  service,
	}
}

// createFakeConfigRepos initializes two Git repositories in local temporary directories, which can
// be used as fake skia-{public,corp}-config repositories in tests.
func createFakeConfigRepos(t *testing.T, ctx context.Context) (fakeSkiaPublicConfig, fakeSkiaCorpConfig *testutils.GitBuilder) {
	// Create two fake "skia-public-config" and "skia-corp-config" Git repos on the local file system
	// (i.e. "git init" two temporary directories).
	fakeSkiaPublicConfig = testutils.GitInit(t, ctx)
	fakeSkiaCorpConfig = testutils.GitInit(t, ctx)

	// Populate fake repositories with a file that will make it easier to tell them apart later on.
	fakeSkiaPublicConfig.Add(ctx, "which-repo.txt", "This is repo skia-public-config!")
	fakeSkiaPublicConfig.Commit(ctx)
	fakeSkiaCorpConfig.Add(ctx, "which-repo.txt", "This is repo skia-corp-config!")
	fakeSkiaCorpConfig.Commit(ctx)

	// Allow repositories to receive pushes.
	fakeSkiaPublicConfig.AcceptPushes(ctx)
	fakeSkiaCorpConfig.AcceptPushes(ctx)

	return
}

// This is intended to be used in tests that do not need to write to disk, but need a
// git.TempCheckout instance to e.g. compute a path into a checkout.
func addFakeConfigRepoCheckouts(g *Goldpushk) {
	fakeSkiaPublicConfigCheckout := &git.TempCheckout{
		GitDir: "/path/to/skia-public-config",
	}
	fakeSkiaCorpConfigCheckout := &git.TempCheckout{
		GitDir: "/path/to/skia-corp-config",
	}
	g.skiaPublicConfigCheckout = fakeSkiaPublicConfigCheckout
	g.skiaCorpConfigCheckout = fakeSkiaCorpConfigCheckout
}

// writeFileIntoRepo creates a file with the given name and contents into a *git.TempCheckout.
func writeFileIntoRepo(t *testing.T, repo *git.TempCheckout, name, contents string) {
	bytes := []byte(contents)
	path := filepath.Join(string(repo.GitDir), name)
	err := ioutil.WriteFile(path, bytes, os.ModePerm)
	assert.NoError(t, err)
}

// hideStdout replaces os.Stdout with a temp file. This hides any output generated by the code under
// test and leads to a less noisy "go test" output.
func hideStdout(t *testing.T) (cleanup func()) {
	// Back up the real stdout.
	stdout := os.Stdout
	cleanup = func() {
		os.Stdout = stdout
	}

	// Replace os.Stdout with a temporary file.
	fakeStdout, err := ioutil.TempFile("", "fake-stdout")
	assert.NoError(t, err)
	os.Stdout = fakeStdout

	return cleanup
}

// fakeStdin fakes user input via stdin. It replaces stdin with a temporary file with the given fake
// input. The returned function should be called at the end of a test to restore the original stdin.
func fakeStdin(t *testing.T, userInput string) (cleanup func()) {
	// Back up stdin and provide a function to restore it later.
	realStdin := os.Stdin
	cleanup = func() {
		os.Stdin = realStdin
	}

	// Create new file to be used as a fake stdin.
	fakeStdin, err := ioutil.TempFile("", "fake-stdin")
	assert.NoError(t, err)

	// Write fake user input.
	_, err = fakeStdin.WriteString(userInput)
	assert.NoError(t, err)

	// Rewind stdin file so that fmt.Scanf() will pick up what we just wrote.
	_, err = fakeStdin.Seek(0, 0)
	assert.NoError(t, err)

	// Replace real stdout with the fake one.
	os.Stdin = fakeStdin

	return cleanup
}

// assertNumCommits asserts that the given Git repository has the given number of commits.
func assertNumCommits(t *testing.T, ctx context.Context, repo *testutils.GitBuilder, n int64) {
	clone, err := git.NewTempCheckout(ctx, repo.RepoUrl())
	defer clone.Delete()
	assert.NoError(t, err)
	actualN, err := clone.NumCommits(ctx)
	assert.Equal(t, n, actualN)
}

// assertRepositoryContainsFileWithContents asserts the presence of a file with the given contents
// in a git repo.
func assertRepositoryContainsFileWithContents(t *testing.T, ctx context.Context, repo *testutils.GitBuilder, filename, expectedContents string) {
	clone, err := git.NewTempCheckout(ctx, repo.RepoUrl())
	assert.NoError(t, err)
	commits, err := clone.RevList(ctx, "master")
	assert.NoError(t, err)
	lastCommit := commits[0]
	actualContents, err := clone.GetFile(ctx, filename, lastCommit)
	assert.NoError(t, err)
	assert.Equal(t, expectedContents, actualContents)
}

// Generated by running:
// $ kubectl get pods -o custom-columns=NAME:.metadata.labels.app,RUNNING_SINCE:.status.containerStatuses[0].state.running.startedAt
const kubectlGetPodsOutput = `NAME                                                   RUNNING_SINCE
fiddler                                                2019-09-26T22:59:31Z
fiddler                                                2019-09-26T22:59:31Z
fiddler                                                2019-09-26T22:59:54Z
<none>                                                 <none>
<none>                                                 <none>
<none>                                                 <none>
gitsync2                                               2019-09-25T18:34:24Z
gitsync2-staging                                       2019-09-25T18:29:42Z
gold-chrome-baselineserver                             2019-09-24T17:57:25Z
gold-chrome-baselineserver                             2019-09-24T17:57:19Z
gold-chrome-baselineserver                             2019-09-24T17:57:33Z
gold-chrome-diffserver                                 2019-09-05T20:53:42Z
gold-chrome-gpu-baselineserver                         2019-09-24T17:55:23Z
gold-chrome-gpu-baselineserver                         2019-09-24T17:55:06Z
gold-chrome-gpu-baselineserver                         2019-09-24T17:55:14Z
gold-chrome-gpu-diffserver                             2019-09-14T05:56:23Z
gold-chrome-gpu-ingestion-bt                           2019-09-24T17:53:24Z
gold-chrome-gpu-skiacorrectness                        2019-09-23T16:42:39Z
gold-chrome-ingestion-bt                               2019-09-24T17:56:10Z
gold-chrome-skiacorrectness                            2019-09-23T16:42:23Z
gold-flutter-baselineserver                            2019-09-24T17:57:32Z
gold-flutter-baselineserver                            <none>
gold-flutter-baselineserver                            2019-09-24T17:57:21Z
gold-flutter-diffserver                                <none>
gold-flutter-engine-baselineserver                     2019-09-24T12:11:35Z
gold-flutter-engine-baselineserver                     2019-09-24T12:11:34Z
gold-flutter-engine-baselineserver                     2019-09-24T12:11:34Z
gold-flutter-engine-diffserver                         2019-09-24T12:10:28Z
gold-flutter-engine-ingestion-bt                       2019-09-24T17:57:45Z
gold-flutter-engine-skiacorrectness                    2019-09-24T12:21:58Z
gold-flutter-ingestion-bt                              2019-09-24T17:59:26Z
gold-flutter-skiacorrectness                           2019-09-23T16:47:49Z
gold-goldpushk-test1-crashing-server                   <none>
gold-goldpushk-test1-healthy-server                    2019-09-26T20:31:44Z
gold-goldpushk-test2-crashing-server                   <none>
gold-goldpushk-test2-healthy-server                    2019-09-26T20:31:45Z
gold-lottie-diffserver                                 2019-09-25T07:36:38Z
gold-lottie-ingestion-bt                               2019-09-24T18:01:03Z
gold-lottie-skiacorrectness                            2019-09-23T16:49:10Z
gold-pdfium-diffserver                                 2019-08-16T15:16:37Z
gold-pdfium-ingestion-bt                               2019-09-25T07:36:14Z
gold-pdfium-skiacorrectness                            2019-09-23T16:49:22Z
gold-skia-diffserver                                   2019-09-05T15:17:16Z
gold-skia-ingestion-bt                                 2019-09-24T18:02:47Z
gold-skia-public-skiacorrectness                       2019-09-24T16:52:42Z
gold-skia-skiacorrectness                              2019-09-24T16:51:49Z
grafana                                                2019-08-28T14:09:11Z
jsdoc                                                  2019-09-20T13:04:44Z
jsdoc                                                  2019-09-20T13:04:38Z
jsfiddle                                               2019-09-26T22:55:01Z
jsfiddle                                               2019-09-26T22:55:10Z
k8s-checker                                            2019-09-22T14:50:26Z
leasing                                                2019-09-12T02:14:12Z
`
