#!/bin/sh
# Purpose: Find relevant sections in this development skill using literal keywords.
# Pattern: Read-only local command over the current Markdown reference files.
# Usage: meca develop "delayed task queue", or sh <skill-root>/scripts/search.sh.
# Related: internal/config/bootstrap/defaults/development/SKILL.md, internal/config/bootstrap/development_search_test.go.
set -eu

usage() {
    printf '%s\n' 'Usage: meca develop "keyword keyword"'
    printf '%s\n' 'Searches local references and returns up to 10 ranked path:line:heading matches.'
}

if [ "$#" -eq 1 ] && { [ "$1" = "--help" ] || [ "$1" = "-h" ]; }; then
    usage
    exit 0
fi
case "$*" in
    *[![:space:]]*) ;;
    *) usage >&2; exit 2 ;;
esac

skill_directory=$(CDPATH= cd -P "$(dirname "$0")/.." && pwd)
if [ ! -d "$skill_directory/references" ] || [ -L "$skill_directory/references" ]; then
    printf '%s\n' 'The skill references must be a real directory beside scripts/.' >&2
    exit 2
fi

# Environment values preserve query backslashes without awk -v escape processing.
MECA_DEVELOPMENT_QUERY=$*
MECA_DEVELOPMENT_ROOT=$skill_directory
export MECA_DEVELOPMENT_QUERY MECA_DEVELOPMENT_ROOT

search_rows=$(find "$skill_directory/references" -type f -name '*.md' -exec awk '
    function finish_section(    count, weight, text, term) {
        if (!section_line) return
        count = 0
        weight = 0
        text = tolower(title " " section)
        for (term in terms) {
            if (seen[term] || index(text, term) || index(tolower(relative_path), term))
                count++
            if (index(text, term)) weight += 2
            if (index(tolower(relative_path), term)) weight++
        }
        if (count > best_count || (count == best_count && weight > best_weight)) {
            best_count = count
            best_weight = weight
            best_line = section_line
            best_heading = section
        }
    }

    function finish_file() {
        finish_section()
        if (best_count > 0)
            printf "%d\t%d\t%s\t%d\t%s\t%d\n", best_count, best_weight,
                relative_path, best_line, best_heading, term_count
    }

    BEGIN {
        size = split(tolower(ENVIRON["MECA_DEVELOPMENT_QUERY"]), words, /[[:space:]]+/)
        for (i = 1; i <= size; i++) {
            if (words[i] != "" && !(words[i] in terms)) {
                terms[words[i]] = 1
                term_count++
            }
        }
        prefix_length = length(ENVIRON["MECA_DEVELOPMENT_ROOT"]) + 1
    }

    FNR == 1 {
        finish_file()
        relative_path = substr(FILENAME, prefix_length + 1)
        title = ""
        section = relative_path
        section_line = 1
        best_count = 0
        best_weight = 0
        fence = ""
        for (term in seen) delete seen[term]
    }

    {
        line = $0
        if (match(line, /^[[:space:]]*(```+|~~~+)/)) {
            marker = substr(line, RSTART, RLENGTH)
            gsub(/[[:space:]]/, "", marker)
            if (fence == "") {
                fence = marker
            } else if (substr(marker, 1, 1) == substr(fence, 1, 1) &&
                       length(marker) >= length(fence) &&
                       substr(line, RSTART + RLENGTH) ~ /^[[:space:]]*$/) {
                fence = ""
            }
            next
        }
        if (fence == "" && line ~ /^#+[[:space:]]/) {
            finish_section()
            section = line
            sub(/^#+[[:space:]]+/, "", section)
            gsub(/[[:space:]]+/, " ", section)
            sub(/[[:space:]]*#+[[:space:]]*$/, "", section)
            if (line ~ /^#[[:space:]]/) title = section
            section_line = FNR
            for (term in seen) delete seen[term]
            next
        }
        text = tolower(line)
        for (term in terms) {
            if (index(text, term)) seen[term] = 1
        }
    }

    END { finish_file() }
' {} +)

if [ -z "$search_rows" ]; then
    printf '%s\n' 'No matching reference. Try concrete English keywords from the task.'
    exit 0
fi

search_ordered=$(printf '%s\n' "$search_rows" | LC_ALL=C sort -t "$(printf '\t')" -k1,1nr -k2,2nr -k3,3)
printf '%s\n' "$search_ordered" | awk -F '\t' '
    NR <= 10 { printf "%s:%s: %s [%d/%d terms]\n", $3, $4, $5, $1, $6 }
'
