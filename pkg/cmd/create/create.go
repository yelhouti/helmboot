package create

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/jenkins-x-labs/helmboot/pkg/common"
	"github.com/jenkins-x-labs/helmboot/pkg/envfactory"
	"github.com/jenkins-x/jx/pkg/cloud"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/step/verify"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"

	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/spf13/cobra"
)

var (
	createLong = templates.LongDesc(`
		Upgrades your Development environments git repository to upgrade versions.

		If the current working directory is already a git clone then the current directory will be modified. 

		Otherwise a new directory will be created by cloning the default helmfile boot git repository and then 
		modifications will be made to replicate the requirements configuration along with the  Environment, Scheduler and SourceRepository resources
`)

	createExample = templates.Examples(`
		# create a new git repository which we can then boot up
		helmboot create
	`)
)

// CreateOptions the options for viewing running PRs
type CreateOptions struct {
	envfactory.EnvFactory
	InitialGitURL string
	Dir           string
	Requirements  config.RequirementsConfig
	Cmd           *cobra.Command
	Args          []string
	Jenkins       bool
}

// NewCmdCreate creates a command object for the "create" command
func NewCmdCreate() (*cobra.Command, *CreateOptions) {
	o := &CreateOptions{}

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Creates a new git repository for a new Jenkins X installation",
		Long:    createLong,
		Example: createExample,
		Run: func(cmd *cobra.Command, args []string) {
			o.Args = args
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.Cmd = cmd

	cmd.Flags().StringVarP(&o.InitialGitURL, "initial-git-url", "", "", "The git URL to clone to fetch the initial set of files for a helm 3 / helmfile based git configuration if this command is not run inside a git clone or against a GitOps based cluster")
	cmd.Flags().BoolVarP(&o.Jenkins, "jenkins", "", false, "Enable the Jenkins Operator for managing Jenkins servers via GitOps")

	AddRequirementsOptions(cmd, &o.Requirements)
	o.EnvFactory.AddFlags(cmd)
	return cmd, o
}

func AddRequirementsOptions(cmd *cobra.Command, r *config.RequirementsConfig) {
	cmd.Flags().StringVarP(&r.BootConfigURL, "boot-config-url", "", "", "specify the boot configuration git URL")

	// auto upgrade
	cmd.Flags().StringVarP(&r.AutoUpdate.Schedule, "autoupdate-schedule", "", "", "the cron schedule for auto upgrading your cluster")

	// cluster
	cmd.Flags().StringVarP(&r.Cluster.ClusterName, "cluster", "c", "", "configures the cluster name")
	cmd.Flags().StringVarP(&r.Cluster.Namespace, "namespace", "n", "", "configures the namespace to use")
	cmd.Flags().StringVarP(&r.Cluster.Provider, "provider", "p", "", "configures the kubernetes provider.  Supported providers: "+cloud.KubernetesProviderOptions())
	cmd.Flags().StringVarP(&r.Cluster.ProjectID, "project", "", "", "configures the Google Project ID")
	cmd.Flags().StringVarP(&r.Cluster.Registry, "registry", "", "", "configures the host name of the container registry")
	cmd.Flags().StringVarP(&r.Cluster.Region, "region", "r", "", "configures the cloud region")
	cmd.Flags().StringVarP(&r.Cluster.Zone, "zone", "z", "", "configures the cloud zone")

	cmd.Flags().StringVarP(&r.Cluster.ExternalDNSSAName, "extdns-sa", "", "", "configures the External DNS service account name")
	cmd.Flags().StringVarP(&r.Cluster.KanikoSAName, "kaniko-sa", "", "", "configures the Kaniko service account name")

	AddGitRequirementsOptions(cmd, r)

	// ingress
	cmd.Flags().StringVarP(&r.Ingress.Domain, "domain", "d", "", "configures the domain name")
	cmd.Flags().StringVarP(&r.Ingress.TLS.Email, "tls-email", "", "", "the TLS email address to enable TLS on the domain")

	// storage
	cmd.Flags().StringVarP(&r.Storage.Logs.URL, "bucket-logs", "", "", "the bucket URL to store logs")
	cmd.Flags().StringVarP(&r.Storage.Backup.URL, "bucket-backups", "", "", "the bucket URL to store backups")
	cmd.Flags().StringVarP(&r.Storage.Repository.URL, "bucket-repo", "", "", "the bucket URL to store repository artifacts")
	cmd.Flags().StringVarP(&r.Storage.Reports.URL, "bucket-reports", "", "", "the bucket URL to store reports. If not specified default to te logs bucket")

	// vault
	cmd.Flags().StringVarP(&r.Vault.Name, "vault-name", "", "", "specify the vault name")
	cmd.Flags().StringVarP(&r.Vault.Bucket, "vault-bucket", "", "", "specify the vault bucket")
	cmd.Flags().StringVarP(&r.Vault.Keyring, "vault-keyring", "", "", "specify the vault key ring")
	cmd.Flags().StringVarP(&r.Vault.Key, "vault-key", "", "", "specify the vault key")
	cmd.Flags().StringVarP(&r.Vault.ServiceAccount, "vault-sa", "", "", "specify the vault Service Account name")

	// velero
	cmd.Flags().StringVarP(&r.Velero.ServiceAccount, "velero-sa", "", "", "specify the Velero Service Account name")
	cmd.Flags().StringVarP(&r.Velero.Namespace, "velero-ns", "", "", "specify the Velero Namespace")

	// version stream
	cmd.Flags().StringVarP(&r.VersionStream.URL, "version-stream-url", "", "", "specify the Version Stream git URL")
	cmd.Flags().StringVarP(&r.VersionStream.Ref, "version-stream-ref", "", "", "specify the Version Stream git reference (branch, tag, sha)")
}

// AddGitRequirementsOptions adds git specific overrides to the given requirements
func AddGitRequirementsOptions(cmd *cobra.Command, r *config.RequirementsConfig) {
	cmd.Flags().StringVarP(&r.Cluster.GitKind, "git-kind", "", "", fmt.Sprintf("the kind of git repository to use. Possible values: %s", strings.Join(gits.KindGits, ", ")))
	cmd.Flags().StringVarP(&r.Cluster.GitName, "git-name", "", "", "the name of the git repository")
	cmd.Flags().StringVarP(&r.Cluster.GitServer, "git-server", "", "", "the git server host such as https://github.com or https://gitlab.com")
	cmd.Flags().StringVarP(&r.Cluster.EnvironmentGitOwner, "env-git-owner", "", "", "the git owner (organisation or user) used to own the git repositories for the environments")
}

// Run implements the command
func (o *CreateOptions) Run() error {
	if o.Gitter == nil {
		o.Gitter = gits.NewGitCLI()
	}

	dir, err := o.gitCloneIfRequired(o.Gitter)
	if err != nil {
		return err
	}

	err = o.overrideRequirements(dir)
	if err != nil {
		return errors.Wrapf(err, "failed to override requirements in dir %s", dir)
	}

	err = o.verifyPreInstall(dir)
	if err != nil {
		return errors.Wrapf(err, "failed to verify requirements in dir %s", dir)
	}

	log.Logger().Infof("created git source at %s", util.ColorInfo(dir))

	err = o.addAndCommitFiles(dir)
	if err != nil {
		return err
	}

	return o.EnvFactory.CreateDevEnvGitRepository(dir)
}

// gitCloneIfRequired if the specified directory is already a git clone then lets just use it
// otherwise lets make a temporary directory and clone the git repository specified
// or if there is none make a new one
func (o *CreateOptions) gitCloneIfRequired(gitter gits.Gitter) (string, error) {
	gitURL := o.InitialGitURL
	if gitURL == "" {
		if o.Jenkins {
			gitURL = common.DefaultJenkinsBootHelmfileRepository
		} else {
			gitURL = common.DefaultBootHelmfileRepository
		}
	}
	var err error
	dir := o.Dir
	if dir == "" {
		dir, err = ioutil.TempDir("", "helmboot-")
		if err != nil {
			return "", errors.Wrap(err, "failed to create temporary directory")
		}
	}

	log.Logger().Infof("cloning %s to directory %s", util.ColorInfo(gitURL), util.ColorInfo(dir))

	err = gitter.Clone(gitURL, dir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to clone repository %s to directory: %s", gitURL, dir)
	}
	return dir, nil
}

func (o *CreateOptions) verifyPreInstall(dir string) error {
	vo := verify.StepVerifyPreInstallOptions{}
	vo.CommonOptions = o.EnvFactory.JXAdapter().NewCommonOptions()
	vo.Dir = dir
	return vo.Run()
}

func (o *CreateOptions) overrideRequirements(dir string) error {
	requirements, fileName, err := config.LoadRequirementsConfig(dir)
	if err != nil {
		return err
	}
	if fileName == "" {
		fileName = filepath.Join(o.Dir, config.RequirementsConfigFileName)
	}
	o.Requirements = *requirements

	// lets re-parse the CLI arguments to re-populate the loaded requirements
	err = o.Cmd.Flags().Parse(o.Args)
	if err != nil {
		return errors.Wrap(err, "failed to reparse arguments")
	}

	err = o.applyDefaults()
	if err != nil {
		return err
	}

	err = o.Requirements.SaveConfig(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", fileName)
	}

	log.Logger().Infof("saved file: %s", util.ColorInfo(fileName))
	return nil
}

func (o *CreateOptions) applyDefaults() error {
	r := &o.Requirements

	gitKind := r.Cluster.GitKind
	gitKinds := append(gits.KindGits, "fake")
	if gitKind != "" && util.StringArrayIndex(gitKinds, gitKind) < 0 {
		return util.InvalidOption("git-kind", gitKind, gits.KindGits)
	}

	// default flags if associated values
	if r.AutoUpdate.Schedule != "" {
		r.AutoUpdate.Enabled = true
	}
	if r.Ingress.TLS.Email != "" {
		r.Ingress.TLS.Enabled = true
	}

	// enable storage if we specify a URL
	storage := &r.Storage
	if storage.Logs.URL != "" && storage.Reports.URL == "" {
		storage.Reports.URL = storage.Logs.URL
	}
	o.defaultStorage(&storage.Backup)
	o.defaultStorage(&storage.Logs)
	o.defaultStorage(&storage.Reports)
	o.defaultStorage(&storage.Repository)
	return nil
}

func (o *CreateOptions) defaultStorage(storage *config.StorageEntryConfig) {
	if storage.URL != "" {
		storage.Enabled = true
	}
}

func (o *CreateOptions) addAndCommitFiles(dir string) error {
	err := o.Gitter.Add(dir, "*")
	if err != nil {
		return errors.Wrapf(err, "failed to add files to git")
	}
	err = o.Gitter.CommitIfChanges(dir, "fix: initial code")
	if err != nil {
		return errors.Wrapf(err, "failed to git commit initial code changes")
	}
	return nil
}
