package shell

import (
	"fmt"
	"strings"
)

func Script(shell string) (string, error) {
	switch strings.ToLower(shell) {
	case "zsh":
		return zshScript, nil
	case "bash":
		return bashScript, nil
	case "fish":
		return fishScript, nil
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

const commonSubcommandGuard = `
    case "$1" in
      scan|list|doctor|trust|init|completion|man|help|version|__resolve|__shell-apply|__browse|-h|--help|-v|--version)
        command wo "$@"
        return $?
        ;;
    esac
`

const zshScript = `# wo shell integration (zsh)
wo() {
` + commonSubcommandGuard + `
  local clean=0
  local pick=0
  local query_parts=()
  local arg
  for arg in "$@"; do
    case "$arg" in
      --clean) clean=1 ;;
      --pick) pick=1 ;;
      *) query_parts+=("$arg") ;;
    esac
  done

  local query="${(j: :)query_parts}"
  if [[ -z "$query" ]]; then
    local json
    local resolve_status
    if [[ $clean -eq 1 ]]; then
      json="$(command wo __browse --clean --json)"
      resolve_status=$?
    else
      json="$(command wo __browse --json)"
      resolve_status=$?
    fi
    if [[ $resolve_status -ne 0 && -z "$json" ]]; then
      return $resolve_status
    fi
    eval "$(printf '%s' "$json" | command wo __shell-apply --shell zsh)"
    return $?
  fi

  local json
  local resolve_status
  if [[ $clean -eq 1 && $pick -eq 1 ]]; then
    json="$(command wo __resolve --query "$query" --clean --pick --json)"
    resolve_status=$?
  elif [[ $clean -eq 1 ]]; then
    json="$(command wo __resolve --query "$query" --clean --json)"
    resolve_status=$?
  elif [[ $pick -eq 1 ]]; then
    json="$(command wo __resolve --query "$query" --pick --json)"
    resolve_status=$?
  else
    json="$(command wo __resolve --query "$query" --json)"
    resolve_status=$?
  fi
  if [[ $resolve_status -ne 0 && -z "$json" ]]; then
    return $resolve_status
  fi

  eval "$(printf '%s' "$json" | command wo __shell-apply --shell zsh)"
  return $?
}
`

const bashScript = `# wo shell integration (bash)
wo() {
` + commonSubcommandGuard + `
  local clean=0
  local pick=0
  local query_parts=()
  local arg
  for arg in "$@"; do
    case "$arg" in
      --clean) clean=1 ;;
      --pick) pick=1 ;;
      *) query_parts+=("$arg") ;;
    esac
  done

  local query="${query_parts[*]}"
  if [[ -z "$query" ]]; then
    local json
    local resolve_status
    if [[ $clean -eq 1 ]]; then
      json="$(command wo __browse --clean --json)"
      resolve_status=$?
    else
      json="$(command wo __browse --json)"
      resolve_status=$?
    fi
    if [[ $resolve_status -ne 0 && -z "$json" ]]; then
      return $resolve_status
    fi
    eval "$(printf '%s' "$json" | command wo __shell-apply --shell bash)"
    return $?
  fi

  local json
  local resolve_status
  if [[ $clean -eq 1 && $pick -eq 1 ]]; then
    json="$(command wo __resolve --query "$query" --clean --pick --json)"
    resolve_status=$?
  elif [[ $clean -eq 1 ]]; then
    json="$(command wo __resolve --query "$query" --clean --json)"
    resolve_status=$?
  elif [[ $pick -eq 1 ]]; then
    json="$(command wo __resolve --query "$query" --pick --json)"
    resolve_status=$?
  else
    json="$(command wo __resolve --query "$query" --json)"
    resolve_status=$?
  fi
  if [[ $resolve_status -ne 0 && -z "$json" ]]; then
    return $resolve_status
  fi

  eval "$(printf '%s' "$json" | command wo __shell-apply --shell bash)"
  return $?
}
`

const fishScript = `# wo shell integration (fish)
function wo --description 'workspace manager'
  if test (count $argv) -gt 0
    set first $argv[1]
    switch $first
      case scan list doctor trust init completion man help version __resolve __shell-apply __browse -h --help -v --version
        command wo $argv
        return $status
    end
  end

  set clean 0
  set pick 0
  set query_parts
  for arg in $argv
    switch $arg
      case --clean
        set clean 1
      case --pick
        set pick 1
      case '*'
        set query_parts $query_parts "$arg"
    end
  end

  set query (string join ' ' $query_parts)
  if test -z "$query"
    set browse_cmd "command wo __browse --json"
    if test $clean -eq 1
      set browse_cmd "$browse_cmd --clean"
    end
    set json (eval $browse_cmd)
    set resolve_status $status
    if test $resolve_status -ne 0 -a -z "$json"
      return $status
    end
    set script (printf '%s' "$json" | command wo __shell-apply --shell fish)
    eval $script
    return $status
  end

  set cmd "command wo __resolve --query \"$query\" --json"
  if test $clean -eq 1
    set cmd "$cmd --clean"
  end
  if test $pick -eq 1
    set cmd "$cmd --pick"
  end

  set json (eval $cmd)
  set resolve_status $status
  if test $resolve_status -ne 0 -a -z "$json"
    return $status
  end

  set script (printf '%s' "$json" | command wo __shell-apply --shell fish)
  eval $script
  return $status
end
`
