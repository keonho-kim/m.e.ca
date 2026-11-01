// Purpose: Define the small command and option vocabulary accepted by LAT queries.
// Pattern: Explicit query grammar.
// Usage: The AST parser validates every literal invocation before rendering it.
// Related: internal/policy/readonly/parse.go, internal/policy/readonly/git.go.
package readonly

import (
	"regexp"
	"strings"
)

func validate(args []string) error {
	name, rest := args[0], args[1:]
	var err error
	switch name {
	case "pwd":
		_, err = options(rest, "LP", "", "", "")
	case "ls":
		_, err = options(rest, "aAlhRtdrF1Ssinp", "", "all almost-all directory human-readable recursive reverse size inode numeric-uid-gid", "")
	case "cat":
		_, err = options(rest, "benstuvAET", "", "number number-nonblank squeeze-blank show-all show-ends show-tabs show-nonprinting", "")
	case "head", "tail":
		_, err = options(rest, "qv", "nc", "quiet silent verbose", "lines bytes")
	case "wc":
		_, err = options(rest, "clmwL", "", "bytes chars lines max-line-length words", "")
	case "grep":
		_, err = options(rest, "EFGHhIilLnoqRrsvVwxcab", "eABCDfm", "extended-regexp fixed-strings basic-regexp ignore-case invert-match word-regexp line-regexp count files-without-match files-with-matches line-number with-filename no-filename only-matching quiet silent recursive dereference-recursive binary-files text byte-offset null line-buffered", "regexp file after-context before-context context directories max-count exclude exclude-dir include")
	case "rg":
		_, err = options(rest, "abcFhHiIlLnNoqRsSstTuUvVwx0", "eAfBgCmjME", "files hidden no-ignore no-ignore-vcs ignore-case case-sensitive smart-case fixed-strings line-regexp word-regexp count count-matches files-with-matches files-without-match line-number no-line-number with-filename no-filename only-matching quiet text invert-match multiline multiline-dotall null null-data follow stats no-messages no-heading heading pretty json passthru no-config", "regexp file glob iglob type type-not after-context before-context context max-count max-depth max-filesize max-columns threads encoding color colors")
	case "uniq":
		var operands []string
		operands, err = options(rest, "cdiuDz", "fsw", "count repeated all-repeated unique ignore-case zero-terminated", "skip-fields skip-chars check-chars")
		if err == nil && len(operands) > 1 {
			err = denied("uniq accepts at most one input file; output files are forbidden")
		}
	case "cut":
		_, err = options(rest, "nsz", "bcdf", "complement only-delimited zero-terminated", "bytes characters fields delimiter output-delimiter")
	case "tr":
		var operands []string
		operands, err = options(rest, "cdsCt", "", "complement delete squeeze-repeats truncate-set1", "")
		if err == nil && (len(operands) < 1 || len(operands) > 2) {
			err = denied("tr requires one or two literal character sets")
		}
	case "du":
		_, err = options(rest, "ahskcmx", "d", "all summarize human-readable total one-file-system", "max-depth")
	case "stat":
		_, err = options(rest, "FLlnqstx", "fc", "dereference file-system terse", "format printf")
	case "readlink":
		_, err = options(rest, "efmnqsvz", "", "canonicalize canonicalize-existing canonicalize-missing no-newline quiet silent verbose zero", "")
	case "basename", "dirname":
		_, err = options(rest, "az", "s", "multiple zero", "suffix")
	case "printf":
		// The standalone host printf only formats stdout; shell expansions are absent.
	case "find":
		err = find(rest)
	case "sed":
		err = sed(rest)
	case "git":
		err = git(rest)
	default:
		err = denied("command %q is unsupported", name)
	}
	return err
}

// options parses flags even after operands, matching getopt implementations that
// permute arguments. A forbidden flag cannot hide behind a filename or pattern.
func options(args []string, flags, values, longFlags, longValues string) ([]string, error) {
	var operands []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return append(operands, args[i+1:]...), nil
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			operands = append(operands, arg)
			continue
		}
		if strings.HasPrefix(arg, "--") {
			key, _, supplied := strings.Cut(arg[2:], "=")
			if member(longFlags, key) && !supplied {
				continue
			}
			if !member(longValues, key) {
				return nil, denied("unsupported option %q", arg)
			}
			if !supplied {
				i++
				if i == len(args) {
					return nil, denied("option %q needs a value", arg)
				}
				if strings.HasPrefix(args[i], "-") {
					return nil, denied("ambiguous value for %q; attach option values with '=' or use an explicit literal path", arg)
				}
			}
			continue
		}
		for j := 1; j < len(arg); j++ {
			flag := arg[j]
			if strings.ContainsRune(values, rune(flag)) {
				if j+1 == len(arg) {
					i++
					if i == len(args) {
						return nil, denied("option %q needs a value", arg)
					}
					if strings.HasPrefix(args[i], "-") {
						return nil, denied("ambiguous value for %q; attach the value to the short option", arg)
					}
				}
				break
			}
			if !strings.ContainsRune(flags, rune(flag)) {
				return nil, denied("unsupported option %q", arg)
			}
		}
	}
	return operands, nil
}

func member(list, value string) bool {
	for _, candidate := range strings.Fields(list) {
		if candidate == value {
			return true
		}
	}
	return false
}

var sedPrint = regexp.MustCompile(`^([0-9]+|\$)?(,([0-9]+|\$))?p$`)

func sed(args []string) error {
	if len(args) > 0 && args[0] == "-n" {
		args = args[1:]
	}
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	if len(args) == 0 || !sedPrint.MatchString(args[0]) {
		return denied("sed supports only an optional -n and a print address such as '1,80p'; scripts and editing flags are forbidden")
	}
	for _, path := range args[1:] {
		if strings.HasPrefix(path, "-") && path != "-" {
			return denied("sed input paths beginning with '-' require an explicit './' path")
		}
	}
	return nil
}

func find(args []string) error {
	predicate := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !predicate && (arg == "-H" || arg == "-L" || arg == "-P") {
			continue
		}
		if arg != "!" && arg != "(" && arg != ")" && !strings.HasPrefix(arg, "-") {
			if predicate {
				return denied("find paths must precede predicates")
			}
			continue
		}
		predicate = true
		if member("! ( ) -a -and -o -or -not -print -print0 -ls -prune -quit -depth -empty -readable -writable -executable", arg) {
			continue
		}
		if !member("-name -iname -path -ipath -wholename -iwholename -regex -iregex -type -size -mtime -mmin -ctime -cmin -atime -amin -newer -maxdepth -mindepth -perm -user -group -links -inum", arg) {
			return denied("unsupported find predicate %q; execution, deletion and file output are forbidden", arg)
		}
		i++
		if i == len(args) {
			return denied("find predicate %q needs a value", arg)
		}
	}
	return nil
}
