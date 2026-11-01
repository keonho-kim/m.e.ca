// Purpose: Limit Git queries and suppress repository-configured execution helpers.
// Pattern: Explicit subcommand grammar and execution normalization.
// Usage: Query validation and rendering share these Git-specific rules.
// Related: internal/policy/readonly/parse.go, internal/tools/local/read_only_command.go.
package readonly

func git(args []string) error {
	if len(args) == 0 {
		return denied("git requires a supported query subcommand")
	}
	var err error
	switch args[0] {
	case "status":
		_, err = options(args[1:], "sbz", "u", "short branch porcelain long verbose ignored", "porcelain untracked-files ignore-submodules")
	case "diff":
		_, err = options(args[1:], "pusrwbaRz", "U", "patch no-patch raw patch-with-raw patch-with-stat numstat shortstat dirstat cumulative check summary name-only name-status stat compact-summary binary full-index color no-color color-words word-diff no-renames find-renames find-copies no-ext-diff no-textconv ignore-space-at-eol ignore-cr-at-eol ignore-blank-lines minimal patience histogram anchored no-index cached staged exit-code quiet relative", "unified inter-hunk-context diff-algorithm word-diff-regex src-prefix dst-prefix line-prefix stat-width stat-name-width stat-count diff-filter")
	case "log", "show":
		_, err = options(args[1:], "pusrwbaRz", "nU", "all branches tags remotes first-parent no-merges merges reverse oneline graph decorate no-decorate date-order author-date-order topo-order patch no-patch raw numstat shortstat summary name-only name-status stat compact-summary full-index color no-color no-ext-diff no-textconv follow root boundary left-right cherry-pick", "max-count skip since until after before author committer grep format pretty date encoding unified abbrev diff-filter")
	case "ls-files":
		_, err = options(args[1:], "zcdmosuikvt", "xX", "cached deleted modified others stage unmerged killed directory no-empty-directory ignored exclude-standard full-name error-unmatch deduplicate eol", "exclude exclude-from exclude-per-directory format")
	case "ls-tree":
		_, err = options(args[1:], "drtzl", "", "name-only name-status full-name full-tree long", "format")
	case "grep":
		_, err = options(args[1:], "EFGhilLnoqvwxzIc", "eABCDfm", "cached no-index untracked no-exclude-standard recurse-submodules text ignore-case invert-match word-regexp line-regexp count files-without-match files-with-matches line-number with-filename no-filename only-matching quiet no-textconv", "regexp file after-context before-context context max-depth threads")
	case "rev-parse":
		_, err = options(args[1:], "q", "", "verify quiet short symbolic symbolic-full-name abbrev-ref show-toplevel show-prefix show-cdup show-object-format is-inside-work-tree is-inside-git-dir is-bare-repository git-dir absolute-git-dir git-common-dir show-superproject-working-tree all branches tags remotes not end-of-options", "short abbrev-ref")
	default:
		err = denied("git subcommand %q is unsupported; use status, diff, log, show, ls-files, ls-tree, grep or rev-parse", args[0])
	}
	return err
}

func gitInvocation(args []string) []string {
	// Command-line configuration outranks local/included repository config. These
	// queries never need pagers, hooks, fsmonitor, external diff or text converters.
	out := []string{args[0], "--no-pager", "--no-optional-locks", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null", "-c", "core.pager=cat", "-c", "diff.external="}
	out = append(out, "-c", "diff.submodule=short", "-c", "status.submoduleSummary=false", "-c", "log.showSignature=false", "-c", "submodule.recurse=false")
	out = append(out, args[1])
	switch args[1] {
	case "diff", "log", "show":
		out = append(out, "--no-ext-diff", "--no-textconv")
	case "grep":
		out = append(out, "--no-textconv")
	}
	return append(out, args[2:]...)
}
