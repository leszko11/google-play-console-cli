package completion

import (
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type node struct {
	path        string
	subcommands []string
	valueFlags  []string
	words       []string
}

func bashScript(root *ffcli.Command) string {
	nodes := buildNodes(root)
	var b strings.Builder
	b.WriteString("# gpc bash completion\n")
	b.WriteString("_gpc_completion_values() {\n")
	b.WriteString("  gpc completion values --flag \"$1\" 2>/dev/null\n")
	b.WriteString("}\n")
	b.WriteString("_gpc_completion() {\n")
	b.WriteString("  local cur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
	b.WriteString("  local prev=\"\"\n")
	b.WriteString("  local path=\"\"\n")
	b.WriteString("  local word=\"\"\n")
	b.WriteString("  local i=1\n")
	b.WriteString("  while [ $i -lt $COMP_CWORD ]; do\n")
	b.WriteString("    word=\"${COMP_WORDS[$i]}\"\n")
	b.WriteString("    if [[ \"$word\" == --*=* ]]; then\n")
	b.WriteString("      i=$((i+1))\n")
	b.WriteString("      continue\n")
	b.WriteString("    fi\n")
	b.WriteString("    if [[ \"$word\" == -* ]]; then\n")
	b.WriteString("      case \"$path|$word\" in\n")
	for _, item := range nodes {
		for _, valueFlag := range item.valueFlags {
			fmt.Fprintf(&b, "        %q) i=$((i+2)); continue ;;\n", item.path+"|"+valueFlag)
		}
	}
	b.WriteString("      esac\n")
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
	b.WriteString("  if [ $COMP_CWORD -gt 0 ]; then\n")
	b.WriteString("    prev=\"${COMP_WORDS[$((COMP_CWORD-1))]}\"\n")
	b.WriteString("  fi\n")
	b.WriteString("  local opts=\"\"\n")
	b.WriteString("  case \"$prev\" in\n")
	b.WriteString("    --package-name) opts=\"$(_gpc_completion_values package-name)\" ;;\n")
	b.WriteString("    --track) opts=\"$(_gpc_completion_values track)\" ;;\n")
	b.WriteString("  esac\n")
	b.WriteString("  if [[ -n \"$opts\" ]]; then\n")
	b.WriteString("    COMPREPLY=( $(compgen -W \"$opts\" -- \"$cur\") )\n")
	b.WriteString("    return\n")
	b.WriteString("  fi\n")
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
	b.WriteString("_gpc_completion_values() {\n")
	b.WriteString("  gpc completion values --flag \"$1\" 2>/dev/null\n")
	b.WriteString("}\n")
	b.WriteString("_gpc_completion() {\n")
	b.WriteString("  local path=\"\"\n")
	b.WriteString("  local word prev\n")
	b.WriteString("  local i=2\n")
	b.WriteString("  while (( i < CURRENT )); do\n")
	b.WriteString("    word=\"${words[i]}\"\n")
	b.WriteString("    if [[ \"$word\" == --*=* ]]; then\n")
	b.WriteString("      ((i++))\n")
	b.WriteString("      continue\n")
	b.WriteString("    fi\n")
	b.WriteString("    if [[ \"$word\" == -* ]]; then\n")
	b.WriteString("      case \"$path|$word\" in\n")
	for _, item := range nodes {
		for _, valueFlag := range item.valueFlags {
			fmt.Fprintf(&b, "        %q) ((i+=2)); continue ;;\n", item.path+"|"+valueFlag)
		}
	}
	b.WriteString("      esac\n")
	b.WriteString("      ((i++))\n")
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
	b.WriteString("    ((i++))\n")
	b.WriteString("  done\n")
	b.WriteString("  if (( CURRENT > 1 )); then\n")
	b.WriteString("    prev=\"${words[CURRENT-1]}\"\n")
	b.WriteString("  fi\n")
	b.WriteString("  local -a opts\n")
	b.WriteString("  case \"$prev\" in\n")
	b.WriteString("    --package-name) opts=(${(f)\"$(_gpc_completion_values package-name)\"}) ;;\n")
	b.WriteString("    --track) opts=(${(f)\"$(_gpc_completion_values track)\"}) ;;\n")
	b.WriteString("  esac\n")
	b.WriteString("  if (( ${#opts[@]} > 0 )); then\n")
	b.WriteString("    compadd -- ${opts[@]}\n")
	b.WriteString("    return\n")
	b.WriteString("  fi\n")
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
	b.WriteString("function __gpc_prev_token_is\n")
	b.WriteString("  set -l tokens (commandline -opc)\n")
	b.WriteString("  if test (count $tokens) -eq 0\n")
	b.WriteString("    return 1\n")
	b.WriteString("  end\n")
	b.WriteString("  test \"$tokens[-1]\" = \"$argv[1]\"\n")
	b.WriteString("end\n")
	b.WriteString("function _gpc_completion_values\n")
	b.WriteString("  gpc completion values --flag $argv[1] 2>/dev/null\n")
	b.WriteString("end\n")
	b.WriteString("complete -c gpc -f\n")
	b.WriteString("complete -c gpc -n \"__gpc_prev_token_is --package-name\" -a \"(_gpc_completion_values package-name)\"\n")
	b.WriteString("complete -c gpc -n \"__gpc_prev_token_is --track\" -a \"(_gpc_completion_values track)\"\n")
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
	rootFlags, rootValueFlags := flagNames(root.FlagSet)
	nodes := make([]node, 0)
	var walk func(path []string, cmd *ffcli.Command)
	walk = func(path []string, cmd *ffcli.Command) {
		words := append([]string{}, rootFlags...)
		valueFlags := append([]string{}, rootValueFlags...)
		cmdFlags, cmdValueFlags := flagNames(cmd.FlagSet)
		words = append(words, cmdFlags...)
		valueFlags = append(valueFlags, cmdValueFlags...)
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
			valueFlags:  uniqueSorted(valueFlags),
			words:       words,
		})
		for _, sub := range cmd.Subcommands {
			walk(append(path, sub.Name), sub)
		}
	}
	walk(nil, root)
	return nodes
}

func flagNames(fs *flag.FlagSet) ([]string, []string) {
	if fs == nil {
		return nil, nil
	}
	names := make([]string, 0)
	valueFlags := make([]string, 0)
	fs.VisitAll(func(f *flag.Flag) {
		name := "--" + f.Name
		names = append(names, name)
		if requiresFlagValue(f) {
			valueFlags = append(valueFlags, name)
		}
	})
	return names, valueFlags
}

type boolFlag interface {
	IsBoolFlag() bool
}

func requiresFlagValue(f *flag.Flag) bool {
	if f == nil || f.Value == nil {
		return false
	}
	bf, ok := f.Value.(boolFlag)
	return !ok || !bf.IsBoolFlag()
}

func zshWords(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, fmt.Sprintf("%q", item))
	}
	return strings.Join(quoted, " ")
}
