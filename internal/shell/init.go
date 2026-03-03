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
      scan|list|doctor|trust|init|completion|man|help|version|__resolve|__shell-apply|__browse|__complete|__completeNoDesc|-h|--help|-v|--version)
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
  local force_global=0
  local positional=()
  local arg
  for arg in "$@"; do
    case "$arg" in
      --clean) clean=1 ;;
      --pick) pick=1 ;;
      --global) force_global=1 ;;
      *) positional+=("$arg") ;;
    esac
  done

  if (( ${#positional[@]} == 0 )); then
    if [[ $force_global -eq 1 ]]; then
      command wo "$@"
      return $?
    fi
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

  if (( ${#positional[@]} > 2 )); then
    command wo "$@"
    return $?
  fi

  local workspace="${positional[1]}"
  local profile=""
  if (( ${#positional[@]} == 2 )); then
    profile="${positional[2]}"
  fi

  local json
  local resolve_status
  local resolve_cmd=(command wo __resolve --query "$workspace" --json)
  if [[ -n "$profile" ]]; then
    resolve_cmd+=(--profile "$profile")
  fi
  if [[ $clean -eq 1 ]]; then
    resolve_cmd+=(--clean)
  fi
  if [[ $pick -eq 1 ]]; then
    resolve_cmd+=(--pick)
  fi
  if [[ $force_global -eq 1 ]]; then
    resolve_cmd+=(--global)
  fi
  json="$("${resolve_cmd[@]}")"
  resolve_status=$?
  if [[ $resolve_status -ne 0 && -z "$json" ]]; then
    return $resolve_status
  fi

  eval "$(printf '%s' "$json" | command wo __shell-apply --shell zsh)"
  return $?
}

if (( ${+functions[compdef]} )); then
  eval "$(command wo completion zsh)"
fi
`

const bashScript = `# wo shell integration (bash)
wo() {
` + commonSubcommandGuard + `
  local clean=0
  local pick=0
  local force_global=0
  local positional=()
  local arg
  for arg in "$@"; do
    case "$arg" in
      --clean) clean=1 ;;
      --pick) pick=1 ;;
      --global) force_global=1 ;;
      *) positional+=("$arg") ;;
    esac
  done

  if [[ ${#positional[@]} -eq 0 ]]; then
    if [[ $force_global -eq 1 ]]; then
      command wo "$@"
      return $?
    fi
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

  if [[ ${#positional[@]} -gt 2 ]]; then
    command wo "$@"
    return $?
  fi

  local workspace="${positional[0]}"
  local profile=""
  if [[ ${#positional[@]} -eq 2 ]]; then
    profile="${positional[1]}"
  fi

  local json
  local resolve_status
  local resolve_cmd=(command wo __resolve --query "$workspace" --json)
  if [[ -n "$profile" ]]; then
    resolve_cmd+=(--profile "$profile")
  fi
  if [[ $clean -eq 1 ]]; then
    resolve_cmd+=(--clean)
  fi
  if [[ $pick -eq 1 ]]; then
    resolve_cmd+=(--pick)
  fi
  if [[ $force_global -eq 1 ]]; then
    resolve_cmd+=(--global)
  fi
  json="$("${resolve_cmd[@]}")"
  resolve_status=$?
  if [[ $resolve_status -ne 0 && -z "$json" ]]; then
    return $resolve_status
  fi

  eval "$(printf '%s' "$json" | command wo __shell-apply --shell bash)"
  return $?
}

if command -v complete >/dev/null 2>&1; then
  source <(command wo completion bash)
fi
`

const fishScript = `# wo shell integration (fish)
function wo --description 'workspace manager'
  if test (count $argv) -gt 0
    set first $argv[1]
    switch $first
      case scan list doctor trust init completion man help version __resolve __shell-apply __browse __complete __completeNoDesc -h --help -v --version
        command wo $argv
        return $status
    end
  end

  set clean 0
  set pick 0
  set force_global 0
  set positional
  for arg in $argv
    switch $arg
      case --clean
        set clean 1
      case --pick
        set pick 1
      case --global
        set force_global 1
      case '*'
        set positional $positional "$arg"
    end
  end

  if test (count $positional) -eq 0
    if test $force_global -eq 1
      command wo $argv
      return $status
    end
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

  if test (count $positional) -gt 2
    command wo $argv
    return $status
  end

  set workspace $positional[1]
  set profile ""
  if test (count $positional) -eq 2
    set profile $positional[2]
  end

  set cmd "command wo __resolve --query \"$workspace\" --json"
  if test -n "$profile"
    set cmd "$cmd --profile \"$profile\""
  end
  if test $clean -eq 1
    set cmd "$cmd --clean"
  end
  if test $pick -eq 1
    set cmd "$cmd --pick"
  end
  if test $force_global -eq 1
    set cmd "$cmd --global"
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

command wo completion fish | source
`
