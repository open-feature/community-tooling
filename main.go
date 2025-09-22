package main

/*
This file is heavily inspired from:
https://github.com/kubernetes/org/blob/main/cmd/merge/main.go

It was the basis for this merger, and some code parts are similar,
but the general approach differs.
*/

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/exp/maps"
	"k8s.io/test-infra/prow/config/org"
	"k8s.io/test-infra/prow/github"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

const (
	Admins      string = "admins"
	Maintainers string = "maintainers"
	Approvers   string = "approvers"
	Triagers    string = "triagers"
	TSC         string = "Technical Steering Committee"
	GB          string = "Governance Board"
)

type options struct {
	config   string
	output   string
	homepage string
}
type Group struct {
	Repos       []string `json:"repos,omitempty"`
	Admins      []string `json:"admins,omitempty"`
	Maintainers []string `json:"maintainers,omitempty"`
	Approvers   []string `json:"approvers,omitempty"`
	Triagers    []string `json:"triagers,omitempty"`
}

func main() {
	o := options{}
	flag.StringVar(&o.config, "config", "config", "")
	flag.StringVar(&o.output, "output", "config/peribolos.yaml", "")
	flag.StringVar(&o.homepage, "homepage", "", "")
	flag.Parse()

	cfg, err := loadOrgs(o)
	if err != nil {
		logrus.Fatalf("Failed to load orgs: %v", err)
	}
	pc := org.FullConfig{
		Orgs: cfg,
	}
	out, err := yaml.Marshal(pc)
	if err != nil {
		logrus.Fatalf("Failed to marshal orgs: %v", err)
	}
	fmt.Println(string(out))

	err = os.WriteFile(o.output, out, 0644)
	if err != nil {
		logrus.Fatalf("Unable to write data into the file: %v", err)
	}
}

func unmarshal(path string) (*org.Config, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %v", err)
	}
	var cfg org.Config
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal: %v", err)
	}
	return &cfg, nil
}

func unmarshalGroup(path string) (*Group, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %v", err)
	}
	var cfg Group
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal: %v", err)
	}
	return &cfg, nil
}

func loadOrgs(o options) (map[string]org.Config, error) {
	config := map[string]org.Config{}
	entries, err := os.ReadDir(o.config)
	if err != nil {
		return nil, fmt.Errorf("error in %s: %v", o.config, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			path := o.config + "/" + name + "/org.yaml"
			cfg, err := unmarshal(path)
			if err != nil {
				return nil, fmt.Errorf("error in %s: %v", path, err)
			}
			if cfg.Teams == nil {
				cfg.Teams = map[string]org.Team{}
			}
			prefix := filepath.Dir(path)
			err = filepath.Walk(prefix, func(path string, info os.FileInfo, err error) error {
				switch {
				case path == prefix:
					return nil // Skip base dir
				case info.IsDir() && filepath.Dir(path) != prefix:
					logrus.Infof("Skipping %s and its children", path)
					return filepath.SkipDir // Skip prefix/foo/bar/ dirs
				case !info.IsDir() && filepath.Dir(path) == prefix:
					return nil // Ignore prefix/foo files
				case filepath.Base(path) == "workgroup.yaml":
					teams, err := generateGroupConfig(path)

					if err != nil {
						return err
					}

					maps.Copy(cfg.Teams, teams)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("merge teams %s: %v", path, err)
			}
			admins := getGlobalTeam(cfg, Admins)
			maintainers := getGlobalTeam(cfg, Maintainers)
			approvers := getGlobalTeam(cfg, Approvers)
			triagers := getGlobalTeam(cfg, Triagers)
			tsc := getGlobalTeam(cfg, TSC)
			gb := getGlobalTeam(cfg, GB)

			for name := range cfg.Repos {
				admins.Repos[name] = github.Admin
				maintainers.Repos[name] = github.Admin
				approvers.Repos[name] = github.Write
				triagers.Repos[name] = github.Triage
				tsc.Repos[name] = github.Admin
				gb.Repos[name] = github.Admin
				cfg.Repos[name] = applyRepoDefaults(cfg, name)

				cfg.Repos[name] = enhanceUrl(cfg, name, o)
			}

			cfg.Teams[Admins] = admins
			cfg.Teams[Maintainers] = maintainers
			cfg.Teams[Approvers] = approvers
			cfg.Teams[Triagers] = triagers
			cfg.Teams[TSC] = tsc
			cfg.Teams[GB] = gb
			config[name] = *cfg
		}
	}
	return config, nil
}

func enhanceUrl(cfg *org.Config, repoName string, o options) org.Repo {
	repo := cfg.Repos[repoName]
	if repo.HomePage == nil && o.homepage != "" {
		homepage := o.homepage
		repo.HomePage = &homepage
	}
	return repo
}
func applyRepoDefaults(cfg *org.Config, repoName string) org.Repo {
	repo := cfg.Repos[repoName]
	true := true
	falsy := false

	if repo.HasProjects == nil {
		repo.HasProjects = &true
	}
	if repo.HasWiki == nil {
		repo.HasWiki = &falsy
	}

	if repo.Archived != nil && *repo.Archived {
		repo.AllowSquashMerge = &falsy
		repo.AllowMergeCommit = &falsy
		repo.AllowRebaseMerge = &falsy
	} else {
		if repo.AllowSquashMerge == nil {
			repo.AllowSquashMerge = &true
		}
		if repo.AllowRebaseMerge == nil {
			repo.AllowRebaseMerge = &falsy
		}
		if repo.AllowMergeCommit == nil {
			repo.AllowMergeCommit = &falsy
		}
	}

	return repo
}

func getGlobalTeam(cfg *org.Config, teamName string) org.Team {
	team, ok := cfg.Teams[teamName]
	if !ok {
		team = org.Team{}
	}
	if team.Repos == nil {
		team.Repos = map[string]github.RepoPermissionLevel{}
	}
	if team.Children == nil {
		team.Children = map[string]org.Team{}
	}
	return team
}

func generateGroupConfig(path string) (map[string]org.Team, error) {
	// Unmarshal the group configuration
	groupCfg, err := unmarshalGroup(path)
	if err != nil {
		return nil, fmt.Errorf("error in %s: %v", path, err)
	}

	group := filepath.Base(filepath.Dir(path))

	// Define the hierarchy of teams
	type TeamHierarchy struct {
		Name     string
		Role     github.RepoPermissionLevel
		Members  []string
		Children []string
	}

	hierarchy := []TeamHierarchy{
		{Name: Admins, Role: github.Admin, Members: groupCfg.Admins},
		{Name: Maintainers, Role: github.Maintain, Members: groupCfg.Maintainers, Children: []string{Admins}},
		{Name: Approvers, Role: github.Write, Members: groupCfg.Approvers, Children: []string{Maintainers}},
		{Name: Triagers, Role: github.Triage, Members: groupCfg.Triagers, Children: []string{Approvers}},
	}

	// Helper function to assign repository permissions
	assignRepoPermissions := func(repos []string, permission github.RepoPermissionLevel) map[string]github.RepoPermissionLevel {
		repoMap := make(map[string]github.RepoPermissionLevel)
		for _, repo := range repos {
			repoMap[repo] = permission
		}
		return repoMap
	}

	// Map to store all teams
	teams := make(map[string]org.Team)

	// Build teams based on the hierarchy
	for _, entry := range hierarchy {
		children := make(map[string]org.Team)
		for _, childName := range entry.Children {
			childKey := group + "-" + childName
			if childTeam, exists := teams[childKey]; exists {
				children[childKey] = childTeam
			}
		}

		teamKey := group + "-" + entry.Name
		teams[teamKey] = org.Team{
			Members:  entry.Members,
			Repos:    assignRepoPermissions(groupCfg.Repos, entry.Role),
			Children: children,
		}
	}

	// Return the top-level team (Triagers) as the root of the hierarchy
	return map[string]org.Team{
		group + "-" + Triagers: teams[group+"-"+Triagers],
	}, nil
}
