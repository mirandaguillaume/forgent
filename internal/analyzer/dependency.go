package analyzer

import (
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/pkg/model"
)

// IssueType represents the type of dependency issue.
type IssueType string

const (
	IssueCircular    IssueType = "circular"
	IssueMissing     IssueType = "missing"
	IssueUnmetContext IssueType = "unmet-context"
)

// DependencyIssue represents a problem found in skill dependencies.
type DependencyIssue struct {
	Type    IssueType
	Skill   string
	Message string
	Details []string
}

// CheckDependencies analyzes a set of skills for dependency issues:
// missing dependencies, circular dependencies, and unmet context.
func CheckDependencies(skills []model.SkillBehavior) []DependencyIssue {
	var issues []DependencyIssue
	skillMap := make(map[string]model.SkillBehavior)
	for _, s := range skills {
		skillMap[s.Skill] = s
	}

	// 1. Check missing dependencies
	for _, skill := range skills {
		for _, dep := range skill.DependsOn {
			if _, ok := skillMap[dep.Skill]; !ok {
				issues = append(issues, DependencyIssue{
					Type:    IssueMissing,
					Skill:   skill.Skill,
					Message: fmt.Sprintf("Depends on %q which does not exist", dep.Skill),
				})
			}
		}
	}

	// 2. Check circular dependencies (DFS cycle detection)
	visited := map[string]bool{}
	inStack := map[string]bool{}

	var dfs func(name string, path []string)
	dfs = func(name string, path []string) {
		if inStack[name] {
			startIdx := indexOf(path, name)
			cycle := append(path[startIdx:], name)
			issues = append(issues, DependencyIssue{
				Type:    IssueCircular,
				Skill:   name,
				Message: fmt.Sprintf("Circular dependency detected: %s", strings.Join(cycle, " -> ")),
				Details: cycle,
			})
			return
		}
		if visited[name] {
			return
		}

		visited[name] = true
		inStack[name] = true

		if skill, ok := skillMap[name]; ok {
			for _, dep := range skill.DependsOn {
				if _, exists := skillMap[dep.Skill]; exists {
					newPath := make([]string, len(path)+1)
					copy(newPath, path)
					newPath[len(path)] = name
					dfs(dep.Skill, newPath)
				}
			}
		}

		inStack[name] = false
	}

	for _, skill := range skills {
		if !visited[skill.Skill] {
			dfs(skill.Skill, nil)
		}
	}

	// 3. Check unmet context
	for _, skill := range skills {
		for _, dep := range skill.DependsOn {
			if depSkill, ok := skillMap[dep.Skill]; ok {
				if !containsString(depSkill.Context.Produces, dep.Provides) {
					issues = append(issues, DependencyIssue{
						Type:    IssueUnmetContext,
						Skill:   skill.Skill,
						Message: fmt.Sprintf("Expects %q from %q, but that skill produces: [%s]", dep.Provides, dep.Skill, strings.Join(depSkill.Context.Produces, ", ")),
					})
				}
			}
		}
	}

	return issues
}

// indexOf returns the index of item in slice, or -1 if not found.
func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

// containsString checks if a string slice contains a given string.
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
