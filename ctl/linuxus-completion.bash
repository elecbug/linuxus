_linuxus_completion() {
    local cur prev commands ps_targets

    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    commands="up down restart ps add-user remove-user clean-volume ensure-disk help"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
        return 0
    fi

    case "${COMP_WORDS[1]}" in
        ps)
            COMPREPLY=( $(compgen -W "container network all c n a" -- "$cur") )
            return 0
            ;;

        add-user|remove-user)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=( $(compgen -W "--user -u" -- "$cur") )
            fi
            return 0
            ;;

        clean-volume|ensure-disk)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=( $(compgen -W "--all -a --user -u" -- "$cur") )
            fi
            return 0
            ;;

        *)
            return 0
            ;;
    esac
}

complete -F _linuxus_completion linuxusctl