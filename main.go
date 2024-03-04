package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/spf13/cobra"
)

type options struct {
	DryRun         bool
	ExistingTags   string
	RepositoryURL  string
	MinimalVersion string
}

func main() {
	options := &options{
		DryRun: true,
	}
	command := &cobra.Command{
		Use:   "krtek",
		Short: "krtek checks if new release of OCP V is available and if yes, creates a new tag and new release",
		Run: func(cmd *cobra.Command, args []string) {
			SearchNewReleases(options)
		},
	}
	command.PersistentFlags().StringVar(&options.MinimalVersion, "minimal-version",
		"", "Do not check versions older than this, expected format: vx.y")
	command.PersistentFlags().StringVar(&options.ExistingTags, "existing-tags",
		"", "list of all existing container tags")
	command.PersistentFlags().StringVar(&options.RepositoryURL, "repository-url",
		"", "url of repository where to check releases")
	command.PersistentFlags().BoolVar(&options.DryRun, "dry-run",
		options.DryRun, "don't create anything")

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}

}
func SearchNewReleases(options *options) {
	minimalVersionConstraint, err := semver.NewConstraint(">= " + options.MinimalVersion)
	if err != nil {
		os.Exit(1)
	}
	filteredOCPVTags, err := filterOldOCPVTags(minimalVersionConstraint, options.ExistingTags)

	repo, err := get_repository(options.RepositoryURL)
	if err != nil {
		fmt.Println("err ")
	}

	pipelinesTasksExistingTags, err := repo.Tags()
	if err != nil {
		fmt.Println("err")
	}
	var filteredPipelinesTasksTags []*semver.Version
	filteredPipelinesTasksTags, err = filterOldPipelinesTasksTags(minimalVersionConstraint, pipelinesTasksExistingTags)

	newTags := getNewTags(filteredOCPVTags, filteredPipelinesTasksTags)
	if len(newTags) > 0 {
		if options.DryRun {
			fmt.Println("DRY RUN enabled - these new tags would be created:")
			for version := range newTags {
				fmt.Println(version)
			}
		} else {
			err := create_tag(repo, newTags)
			if err != nil {
				fmt.Println("err during creation of new tags")
			}
		}
	} else {
		fmt.Println("Nothing to do")
	}
}

func create_tag(repo *git.Repository, newTags map[string]*semver.Version) error {
	for _, tag := range newTags {
		_, err := repo.CreateTag(tag.String(), plumbing.NewHash(tag.String()), &git.CreateTagOptions{Tagger: &object.Signature{
			Name:  "OpenShift Virtualization Maintainers",
			Email: "kubevirt-tekton-tasks@redhat.com",
			When:  time.Now(),
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func getNewTags(oCPVTags, pTTags []*semver.Version) map[string]*semver.Version {
	newTags := map[string]*semver.Version{}
	for _, oCPVTag := range oCPVTags {
		found := false
		for _, pTTag := range pTTags {
			if oCPVTag.Equal(pTTag) {
				found = true
			}
		}
		if !found {
			newTags[oCPVTag.String()] = oCPVTag
		}
	}
	return newTags
}

func filterOldPipelinesTasksTags(minimalVersionConstraint *semver.Constraints, existingPTTags storer.ReferenceIter) ([]*semver.Version, error) {
	existingTags := make([]*semver.Version, 0)
	existingPTTags.ForEach(func(tag *plumbing.Reference) error {
		version, err := semver.NewVersion(tag.Name().Short())
		if err != nil {
			return nil
		}
		if minimalVersionConstraint.Check(version) {
			existingTags = append(existingTags, version)
		}
		return nil
	})
	return existingTags, nil
}

func filterOldOCPVTags(minimalVersionConstraint *semver.Constraints, existingTagsStr string) ([]*semver.Version, error) {
	tags := strings.Split(existingTagsStr, ",")
	highestPatchOfMinorMap := map[string]*semver.Version{}
	existingTags := make([]*semver.Version, 0)
	for _, tag := range tags {
		version, err := semver.NewVersion(tag)
		if err != nil {
			continue
		}
		if version.Prerelease() != "" {
			continue
		}

		majorMinorVersion := fmt.Sprintf("%v.%v", version.Major(), version.Minor())
		if highestVersion, ok := highestPatchOfMinorMap[majorMinorVersion]; ok {
			if highestVersion.Patch() < version.Minor() {
				highestPatchOfMinorMap[majorMinorVersion] = version
			}
		} else {
			highestPatchOfMinorMap[majorMinorVersion] = version
		}
	}
	for _, version := range highestPatchOfMinorMap {
		if minimalVersionConstraint.Check(version) {
			existingTags = append(existingTags, version)
		}
	}
	return existingTags, nil
}

func get_repository(url string) (*git.Repository, error) {
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: url,
	})

	if err != nil {
		return nil, err
	}
	return repo, nil
}
