package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/CircleCI-Public/circleci-cli/api"
	"github.com/CircleCI-Public/circleci-cli/client"
	"github.com/CircleCI-Public/circleci-cli/logger"
	"github.com/CircleCI-Public/circleci-cli/references"
	"github.com/CircleCI-Public/circleci-cli/settings"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

type orbOptions struct {
	apiOpts api.Options
	cfg     *settings.Config
	args    []string
}

var orbAnnotations = map[string]string{
	"<path>":      "The path to your orb (use \"-\" for STDIN)",
	"<namespace>": "The namespace used for the orb (i.e. circleci)",
	"<orb>":       "A fully-qualified reference to an orb. This takes the form namespace/orb@version",
}

var orbListUncertified bool
var orbListJSON bool
var orbListDetails bool

func newOrbCommand(config *settings.Config) *cobra.Command {
	opts := orbOptions{
		apiOpts: api.Options{},
		cfg:     config,
	}

	listCommand := &cobra.Command{
		Use:   "list <namespace>",
		Short: "List orbs",
		Args:  cobra.MaximumNArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			opts.args = args
			opts.apiOpts.Context = context.Background()
			opts.apiOpts.Log = logger.NewLogger(config.Debug)
			opts.apiOpts.Client = client.NewClient(config.Host, config.Endpoint, config.Token)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return listOrbs(opts)
		},
		Annotations: make(map[string]string),
	}
	listCommand.Annotations["<namespace>"] = orbAnnotations["<namespace>"] + " (Optional)"
	listCommand.PersistentFlags().BoolVarP(&orbListUncertified, "uncertified", "u", false, "include uncertified orbs")
	listCommand.PersistentFlags().BoolVar(&orbListJSON, "json", false, "print output as json instead of human-readable")
	listCommand.PersistentFlags().BoolVarP(&orbListDetails, "details", "d", false, "output all the commands, executors, and jobs, along with a tree of their parameters")
	if err := listCommand.PersistentFlags().MarkHidden("json"); err != nil {
		panic(err)
	}

	validateCommand := &cobra.Command{
		Use:   "validate <path>",
		Short: "Validate an orb.yml",
		PreRun: func(cmd *cobra.Command, args []string) {
			opts.args = args
			opts.apiOpts.Context = context.Background()
			opts.apiOpts.Log = logger.NewLogger(config.Debug)
			opts.apiOpts.Client = client.NewClient(config.Host, config.Endpoint, config.Token)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return validateOrb(opts)
		},
		Args:        cobra.ExactArgs(1),
		Annotations: make(map[string]string),
	}
	validateCommand.Annotations["<path>"] = orbAnnotations["<path>"]

	processCommand := &cobra.Command{
		Use:   "process <path>",
		Short: "Validate an orb and print its form after all pre-registration processing",
		PreRun: func(cmd *cobra.Command, args []string) {
			opts.args = args
			opts.apiOpts.Context = context.Background()
			opts.apiOpts.Log = logger.NewLogger(config.Debug)
			opts.apiOpts.Client = client.NewClient(config.Host, config.Endpoint, config.Token)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return processOrb(opts)
		},
		Args:        cobra.ExactArgs(1),
		Annotations: make(map[string]string),
	}
	processCommand.Annotations["<path>"] = orbAnnotations["<path>"]

	publishCommand := &cobra.Command{
		Use:   "publish <path> <orb>",
		Short: "Publish an orb to the registry",
		Long: `Publish an orb to the registry.
Please note that at this time all orbs published to the registry are world-readable.`,
		PreRun: func(cmd *cobra.Command, args []string) {
			opts.args = args
			opts.apiOpts.Context = context.Background()
			opts.apiOpts.Log = logger.NewLogger(config.Debug)
			opts.apiOpts.Client = client.NewClient(config.Host, config.Endpoint, config.Token)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return publishOrb(opts)
		},
		Args:        cobra.ExactArgs(2),
		Annotations: make(map[string]string),
	}
	publishCommand.Annotations["<orb>"] = orbAnnotations["<orb>"]
	publishCommand.Annotations["<path>"] = orbAnnotations["<path>"]

	promoteCommand := &cobra.Command{
		Use:   "promote <orb> <segment>",
		Short: "Promote a development version of an orb to a semantic release",
		Long: `Promote a development version of an orb to a semantic release.
Please note that at this time all orbs promoted within the registry are world-readable.

Example: 'circleci orb publish promote foo/bar@dev:master major' => foo/bar@1.0.0`,
		PreRun: func(cmd *cobra.Command, args []string) {
			opts.args = args
			opts.apiOpts.Context = context.Background()
			opts.apiOpts.Log = logger.NewLogger(config.Debug)
			opts.apiOpts.Client = client.NewClient(config.Host, config.Endpoint, config.Token)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return promoteOrb(opts)
		},
		Args:        cobra.ExactArgs(2),
		Annotations: make(map[string]string),
	}
	promoteCommand.Annotations["<orb>"] = orbAnnotations["<orb>"]
	promoteCommand.Annotations["<segment>"] = `"major"|"minor"|"patch"`

	incrementCommand := &cobra.Command{
		Use:   "increment <path> <namespace>/<orb> <segment>",
		Short: "Increment a released version of an orb",
		Long: `Increment a released version of an orb.
Please note that at this time all orbs incremented within the registry are world-readable.

Example: 'circleci orb publish increment foo/orb.yml foo/bar minor' => foo/bar@1.1.0`,
		PreRun: func(cmd *cobra.Command, args []string) {
			opts.args = args
			opts.apiOpts.Context = context.Background()
			opts.apiOpts.Log = logger.NewLogger(config.Debug)
			opts.apiOpts.Client = client.NewClient(config.Host, config.Endpoint, config.Token)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return incrementOrb(opts)
		},
		Args:        cobra.ExactArgs(3),
		Annotations: make(map[string]string),
		Aliases:     []string{"inc"},
	}
	incrementCommand.Annotations["<path>"] = orbAnnotations["<path>"]
	incrementCommand.Annotations["<segment>"] = `"major"|"minor"|"patch"`

	publishCommand.AddCommand(promoteCommand)
	publishCommand.AddCommand(incrementCommand)

	sourceCommand := &cobra.Command{
		Use:   "source <orb>",
		Short: "Show the source of an orb",
		PreRun: func(cmd *cobra.Command, args []string) {
			opts.args = args
			opts.apiOpts.Context = context.Background()
			opts.apiOpts.Log = logger.NewLogger(config.Debug)
			opts.apiOpts.Client = client.NewClient(config.Host, config.Endpoint, config.Token)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return showSource(opts)
		},
		Args:        cobra.ExactArgs(1),
		Annotations: make(map[string]string),
	}
	sourceCommand.Annotations["<orb>"] = orbAnnotations["<orb>"]
	sourceCommand.Example = `  circleci orb source circleci/python@0.1.4 # grab the source at version 0.1.4
  circleci orb source my-ns/foo-orb@dev:latest # grab the source of dev release "latest"`

	orbInfoCmd := &cobra.Command{
		Use:   "info <orb>",
		Short: "Show the meta-data of an orb",
		PreRun: func(cmd *cobra.Command, args []string) {
			opts.args = args
			opts.apiOpts.Context = context.Background()
			opts.apiOpts.Log = logger.NewLogger(config.Debug)
			opts.apiOpts.Client = client.NewClient(config.Host, config.Endpoint, config.Token)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return orbInfo(opts)
		},
		Args:        cobra.ExactArgs(1),
		Annotations: make(map[string]string),
	}
	orbInfoCmd.Annotations["<orb>"] = orbAnnotations["<orb>"]
	orbInfoCmd.Example = `  circleci orb info circleci/python@0.1.4
  circleci orb info my-ns/foo-orb@dev:latest`

	orbCreate := &cobra.Command{
		Use:   "create <namespace>/<orb>",
		Short: "Create an orb in the specified namespace",
		Long: `Create an orb in the specified namespace
Please note that at this time all orbs created in the registry are world-readable.`,
		PreRun: func(cmd *cobra.Command, args []string) {
			opts.args = args
			opts.apiOpts.Context = context.Background()
			opts.apiOpts.Log = logger.NewLogger(config.Debug)
			opts.apiOpts.Client = client.NewClient(config.Host, config.Endpoint, config.Token)
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return createOrb(opts)
		},
		Args: cobra.ExactArgs(1),
	}

	orbCommand := &cobra.Command{
		Use:   "orb",
		Short: "Operate on orbs",
	}

	orbCommand.AddCommand(listCommand)
	orbCommand.AddCommand(orbCreate)
	orbCommand.AddCommand(validateCommand)
	orbCommand.AddCommand(processCommand)
	orbCommand.AddCommand(publishCommand)
	orbCommand.AddCommand(sourceCommand)
	orbCommand.AddCommand(orbInfoCmd)

	return orbCommand
}

func parameterDefaultToString(parameter api.OrbElementParameter) string {
	defaultValue := " (default: '"

	// If there isn't a default or the default value is for a steps parameter
	// then just ignore the value.
	// It's possible to have a very large list of steps that pollutes the output.
	if parameter.Default == nil || parameter.Type == "steps" {
		return ""
	}

	switch parameter.Type {
	case "enum":
		defaultValue += parameter.Default.(string)
	case "string":
		defaultValue += parameter.Default.(string)
	case "boolean":
		defaultValue += fmt.Sprintf("%t", parameter.Default.(bool))
	default:
		defaultValue += ""
	}

	return defaultValue + "')"
}

func addOrbElementParametersToBuffer(buf *bytes.Buffer, orbElement api.OrbElement) error {
	for parameterName, parameter := range orbElement.Parameters {
		var err error

		defaultValueString := parameterDefaultToString(parameter)
		_, err = buf.WriteString(fmt.Sprintf("       - %s: %s%s\n", parameterName, parameter.Type, defaultValueString))

		if err != nil {
			return err
		}
	}

	return nil
}

func addOrbElementsToBuffer(buf *bytes.Buffer, name string, namedOrbElements map[string]api.OrbElement) {
	var err error

	if len(namedOrbElements) > 0 {
		_, err = buf.WriteString(fmt.Sprintf("  %s:\n", name))
		for elementName, orbElement := range namedOrbElements {
			parameterCount := len(orbElement.Parameters)

			_, err = buf.WriteString(fmt.Sprintf("    - %s: %d parameter(s)\n", elementName, parameterCount))

			if parameterCount > 0 {
				err = addOrbElementParametersToBuffer(buf, orbElement)
			}
		}
	}

	// This will never occur. The docs for bytes.Buffer.WriteString says err
	// will always be nil. The linter still expects this error to be checked.
	if err != nil {
		panic(err)
	}
}

func orbToDetailedString(orb api.OrbWithData) string {
	buffer := bytes.NewBufferString(orbToSimpleString(orb))

	addOrbElementsToBuffer(buffer, "Commands", orb.Commands)
	addOrbElementsToBuffer(buffer, "Jobs", orb.Jobs)
	addOrbElementsToBuffer(buffer, "Executors", orb.Executors)

	return buffer.String()
}

func orbToSimpleString(orb api.OrbWithData) string {
	var buffer bytes.Buffer

	_, err := buffer.WriteString(fmt.Sprintln(orb.Name, "("+orb.HighestVersion+")"))
	if err != nil {
		// The WriteString docstring says that it will never return an error
		panic(err)
	}

	return buffer.String()
}

func orbCollectionToString(orbCollection *api.OrbsForListing) (string, error) {
	var result string

	if orbListJSON {
		orbJSON, err := json.MarshalIndent(orbCollection, "", "  ")
		if err != nil {
			return "", errors.Wrapf(err, "Failed to convert to convert to JSON")
		}
		result = string(orbJSON)
	} else {
		result += fmt.Sprintf("Orbs found: %d. ", len(orbCollection.Orbs))
		if orbListUncertified {
			result += "Includes all certified and uncertified orbs.\n\n"
		} else {
			result += "Showing only certified orbs. Add -u for a list of all orbs.\n\n"
		}
		for _, orb := range orbCollection.Orbs {
			if orbListDetails {
				result += (orbToDetailedString(orb))
			} else {
				result += (orbToSimpleString(orb))
			}
		}
	}

	return result, nil
}

func logOrbs(logger *logger.Logger, orbCollection *api.OrbsForListing) error {
	result, err := orbCollectionToString(orbCollection)
	if err != nil {
		return err
	}

	logger.Info(result)

	return nil
}

func listOrbs(opts orbOptions) error {
	if len(opts.args) != 0 {
		return listNamespaceOrbs(opts)
	}

	orbs, err := api.ListOrbs(opts.apiOpts, orbListUncertified)
	if err != nil {
		return errors.Wrapf(err, "Failed to list orbs")
	}

	return logOrbs(opts.apiOpts.Log, orbs)
}

func listNamespaceOrbs(opts orbOptions) error {
	namespace := opts.args[0]

	orbs, err := api.ListNamespaceOrbs(opts.apiOpts, namespace)
	if err != nil {
		return errors.Wrapf(err, "Failed to list orbs in namespace `%s`", namespace)
	}

	return logOrbs(opts.apiOpts.Log, orbs)
}

func validateOrb(opts orbOptions) error {
	_, err := api.OrbQuery(opts.apiOpts, opts.args[0])

	if err != nil {
		return err
	}

	if opts.args[0] == "-" {
		opts.apiOpts.Log.Infof("Orb input is valid.")
	} else {
		opts.apiOpts.Log.Infof("Orb at `%s` is valid.", opts.args[0])
	}

	return nil
}

func processOrb(opts orbOptions) error {
	response, err := api.OrbQuery(opts.apiOpts, opts.args[0])

	if err != nil {
		return err
	}

	opts.apiOpts.Log.Info(response.OutputYaml)
	return nil
}

func publishOrb(opts orbOptions) error {
	path := opts.args[0]
	ref := opts.args[1]
	namespace, orb, version, err := references.SplitIntoOrbNamespaceAndVersion(ref)
	log := opts.apiOpts.Log

	if err != nil {
		return err
	}

	id, err := api.OrbID(opts.apiOpts, namespace, orb)
	if err != nil {
		return err
	}

	_, err = api.OrbPublishByID(opts.apiOpts, path, id.Orb.ID, version)
	if err != nil {
		return err
	}

	log.Infof("Orb `%s` was published.", ref)
	log.Info("Please note that this is an open orb and is world-readable.")

	if references.IsDevVersion(version) {
		log.Infof("Note that your dev label `%s` can be overwritten by anyone in your organization.", version)
		log.Infof("Your dev orb will expire in 90 days unless a new version is published on the label `%s`.", version)
	}
	return nil
}

var validSegments = map[string]bool{
	"major": true,
	"minor": true,
	"patch": true}

func validateSegmentArg(label string) error {
	if _, valid := validSegments[label]; valid {
		return nil
	}
	return fmt.Errorf("expected `%s` to be one of \"major\", \"minor\", or \"patch\"", label)
}

func incrementOrb(opts orbOptions) error {
	ref := opts.args[1]
	segment := opts.args[2]

	if err := validateSegmentArg(segment); err != nil {
		return err
	}

	namespace, orb, err := references.SplitIntoOrbAndNamespace(ref)
	if err != nil {
		return err
	}

	response, err := api.OrbIncrementVersion(opts.apiOpts, opts.args[0], namespace, orb, segment)

	if err != nil {
		return err
	}

	opts.apiOpts.Log.Infof("Orb `%s` has been incremented to `%s/%s@%s`.\n", ref, namespace, orb, response.HighestVersion)
	opts.apiOpts.Log.Info("Please note that this is an open orb and is world-readable.")
	return nil
}

func promoteOrb(opts orbOptions) error {
	ref := opts.args[0]
	segment := opts.args[1]

	if err := validateSegmentArg(segment); err != nil {
		return err
	}

	namespace, orb, version, err := references.SplitIntoOrbNamespaceAndVersion(ref)
	if err != nil {
		return err
	}

	if !references.IsDevVersion(version) {
		return fmt.Errorf("The version '%s' must be a dev version (the string should begin `dev:`)", version)
	}

	response, err := api.OrbPromote(opts.apiOpts, namespace, orb, version, segment)
	if err != nil {
		return err
	}

	opts.apiOpts.Log.Infof("Orb `%s` was promoted to `%s/%s@%s`.\n", ref, namespace, orb, response.HighestVersion)
	opts.apiOpts.Log.Info("Please note that this is an open orb and is world-readable.")
	return nil
}

func createOrb(opts orbOptions) error {
	var err error

	namespace, orb, err := references.SplitIntoOrbAndNamespace(opts.args[0])

	if err != nil {
		return err
	}

	_, err = api.CreateOrb(opts.apiOpts, namespace, orb)

	if err != nil {
		return err
	}

	opts.apiOpts.Log.Infof("Orb `%s` created.\n", opts.args[0])
	opts.apiOpts.Log.Info("Please note that any versions you publish of this orb are world-readable.\n")
	opts.apiOpts.Log.Infof("You can now register versions of `%s` using `circleci orb publish`.\n", opts.args[0])
	return nil
}

func showSource(opts orbOptions) error {
	ref := opts.args[0]

	source, err := api.OrbSource(opts.apiOpts, ref)
	if err != nil {
		return errors.Wrapf(err, "Failed to get source for '%s'", ref)
	}
	opts.apiOpts.Log.Info(source)
	return nil
}

func orbInfo(opts orbOptions) error {
	ref := opts.args[0]
	log := opts.apiOpts.Log

	info, err := api.OrbInfo(opts.apiOpts, ref)
	if err != nil {
		return errors.Wrapf(err, "Failed to get info for '%s'", ref)
	}

	log.Info("\n")

	if len(info.Orb.Versions) > 0 {
		log.Infof("Latest: %s@%s", info.Orb.Name, info.Orb.HighestVersion)
		log.Infof("Last-updated: %s", info.Orb.Versions[0].CreatedAt)
		log.Infof("Created: %s", info.Orb.CreatedAt)
		firstRelease := info.Orb.Versions[len(info.Orb.Versions)-1]
		log.Infof("First-release: %s @ %s", firstRelease.Version, firstRelease.CreatedAt)

		log.Infof("Total-revisions: %d", len(info.Orb.Versions))
	} else {
		log.Infof("This orb hasn't published any versions yet.")
	}

	log.Info("\n")

	log.Infof("Total-commands: %d", len(info.Orb.Commands))
	log.Infof("Total-executors: %d", len(info.Orb.Executors))
	log.Infof("Total-jobs: %d", len(info.Orb.Jobs))

	return nil
}
