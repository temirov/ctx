package githubdoc

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ruleFile struct {
	IncludeExtensions []string `yaml:"include_exts"`
	SkipDirectory     string   `yaml:"skip_dir_pattern"`
	DropExpressions   []string `yaml:"drop_line_regexps"`
	Content           struct {
		StripLeadingH1 *bool `yaml:"strip_leading_h1"`
		BumpHeadingsBy *int  `yaml:"bump_headings_by"`
	} `yaml:"content"`
}

func LoadRuleSet(path string) (RuleSet, error) {
	if path == "" {
		return RuleSet{}, nil
	}
	data, readErr := os.ReadFile(path)
	if readErr != nil {
		return RuleSet{}, fmt.Errorf("read rules from %s: %w", path, readErr)
	}
	var raw ruleFile
	if unmarshalErr := yaml.Unmarshal(data, &raw); unmarshalErr != nil {
		return RuleSet{}, fmt.Errorf("parse rules from %s: %w", path, unmarshalErr)
	}
	ruleSet := RuleSet{
		IncludeExtensions: raw.IncludeExtensions,
		SkipDirectory:     raw.SkipDirectory,
		DropExpressions:   raw.DropExpressions,
	}
	if raw.Content.StripLeadingH1 != nil {
		ruleSet.StripLeadingH1 = *raw.Content.StripLeadingH1
	}
	if raw.Content.BumpHeadingsBy != nil {
		ruleSet.BumpHeadingsBy = *raw.Content.BumpHeadingsBy
	}
	return ruleSet, nil
}
