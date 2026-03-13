package completion

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type node struct {
	path        string
	subcommands []string
	words       []string
}

func bashScript(root *ffcli.Command) string {
	nodes := buildNodes(root)
	var b strings.Builder
	b.WriteString("# gpc bash completion\n")
	b.WriteString("_gpc_completion() {\n")
	b.WriteString("  local cur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
	b.WriteString("  local path=\"\"\n")
	b.WriteString("  local word=\"\"\n")
	b.WriteString("  local i=1\n")
	b.WriteString("  while [ $i -lt $COMP_CWORD ]; do\n")
	b.WriteString("    word=\"${COMP_WORDS[$i]}\"\n")
	b.WriteString("    if [[ \"$word\" == -* ]]; then\n")
	b.WriteString("      i=$((i+1))\n")
	b.WriteString("      continue\n")
	b.WriteString("    fi\n")
	b.WriteString("    case \"$path|$word\" in\n")
	for _, item := range nodes {
		for _, sub := range item.subcommands {
			fmt.Fprintf(&b, "      %q) path=%q ;;\n", item.path+"|"+sub, strings.TrimSpace(strings.TrimSpace(item.path+" "+sub)))
		}
	}
	b.WriteString("      *) ;;\n")
	b.WriteString("    esac\n")
	b.WriteString("    i=$((i+1))\n")
	b.WriteString("  done\n")
	b.WriteString("  local opts=\"\"\n")
	b.WriteString("  case \"$path\" in\n")
	for _, item := range nodes {
		fmt.Fprintf(&b, "    %q) opts=%q ;;\n", item.path, strings.Join(item.words, " "))
	}
	b.WriteString("  esac\n")
	b.WriteString("  COMPREPLY=( $(compgen -W \"$opts\" -- \"$cur\") )\n")
	b.WriteString("}\n")
	b.WriteString("complete -F _gpc_completion gpc\n")
	return b.String()
}

func zshScript(root *ffcli.Command) string {
	nodes := buildNodes(root)
	var b strings.Builder
	b.WriteString("#compdef gpc\n")
	b.WriteString("_gpc_completion() {\n")
	b.WriteString("  local path=\"\"\n")
	b.WriteString("  local -a args\n")
	b.WriteString("  args=(${words[@]:1})\n")
	b.WriteString("  for word in ${args[@]}; do\n")
	b.WriteString("    [[ \"$word\" == -* ]] && continue\n")
	b.WriteString("    case \"$path|$word\" in\n")
	for _, item := range nodes {
		for _, sub := range item.subcommands {
			fmt.Fprintf(&b, "      %q) path=%q ;;\n", item.path+"|"+sub, strings.TrimSpace(strings.TrimSpace(item.path+" "+sub)))
		}
	}
	b.WriteString("      *) ;;\n")
	b.WriteString("    esac\n")
	b.WriteString("  done\n")
	b.WriteString("  local -a opts\n")
	b.WriteString("  case \"$path\" in\n")
	for _, item := range nodes {
		fmt.Fprintf(&b, "    %q) opts=(%s) ;;\n", item.path, zshWords(item.words))
	}
	b.WriteString("  esac\n")
	b.WriteString("  compadd -- ${opts[@]}\n")
	b.WriteString("}\n")
	b.WriteString("compdef _gpc_completion gpc\n")
	return b.String()
}

func fishScript(root *ffcli.Command) string {
	nodes := buildNodes(root)
	var b strings.Builder
	b.WriteString("# gpc fish completion\n")
	b.WriteString("complete -c gpc -f\n")
	for _, item := range nodes {
		condition := "__fish_use_subcommand"
		if item.path != "" {
			condition = "__fish_seen_subcommand_from " + strings.ReplaceAll(item.path, " ", " ")
		}
		for _, word := range item.words {
			if strings.HasPrefix(word, "--") {
				fmt.Fprintf(&b, "complete -c gpc -n %q -l %s\n", condition, strings.TrimPrefix(word, "--"))
				continue
			}
			fmt.Fprintf(&b, "complete -c gpc -n %q -a %q\n", condition, word)
		}
	}
	return b.String()
}

func buildNodes(root *ffcli.Command) []node {
	if root == nil {
		return nil
	}
	rootFlags := flagNames(root.FlagSet)
	nodes := make([]node, 0)
	var walk func(path []string, cmd *ffcli.Command)
	walk = func(path []string, cmd *ffcli.Command) {
		words := append([]string{}, rootFlags...)
		words = append(words, flagNames(cmd.FlagSet)...)
		subcommands := make([]string, 0, len(cmd.Subcommands))
		for _, sub := range cmd.Subcommands {
			subcommands = append(subcommands, sub.Name)
			words = append(words, sub.Name)
		}
		words = uniqueSorted(words)
		subcommands = uniqueSorted(subcommands)
		nodes = append(nodes, node{
			path:        strings.Join(path, " "),
			subcommands: subcommands,
			words:       words,
		})
		for _, sub := range cmd.Subcommands {
			walk(append(path, sub.Name), sub)
		}
	}
	walk(nil, root)
	return nodes
}

func flagNames(fs *flag.FlagSet) []string {
	if fs == nil {
		return nil
	}
	names := make([]string, 0)
	fs.VisitAll(func(f *flag.Flag) {
		names = append(names, "--"+f.Name)
	})
	return names
}

func uniqueSorted(items []string) []string {
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			continue
		}
		set[item] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func zshWords(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, fmt.Sprintf("%q", item))
	}
	return strings.Join(quoted, " ")
}
