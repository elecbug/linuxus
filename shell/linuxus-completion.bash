_linuxus_completion() {
    local cur prev commands ps_targets

    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    commands="init config-check doctor systemd-unit list-users templates lock-user unlock-user reset-password disconnect-user recover-user assign-template assign-class backup-user restore-user verify-backup up down restart ps add-user remove-user clean-volume ensure-disk help"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
        return 0
    fi

    case "${COMP_WORDS[1]}" in
        ps)
            COMPREPLY=( $(compgen -W "container network all c n a" -- "$cur") )
            return 0
            ;;

        add-user|remove-user|lock-user|unlock-user|reset-password|disconnect-user|recover-user)
            if [[ "$cur" == -* ]]; then
                COMPREPLY=( $(compgen -W "--user -u" -- "$cur") )
            fi
            return 0
            ;;

        backup-user|restore-user|verify-backup|assign-template|assign-class)
            if [[ "$prev" == --file || "$prev" == --output ]]; then
                COMPREPLY=( $(compgen -f -- "$cur") )
            elif [[ "$cur" == -* ]]; then
                case "${COMP_WORDS[1]}" in
                    backup-user) COMPREPLY=( $(compgen -W "--user --output" -- "$cur") ) ;;
                    restore-user) COMPREPLY=( $(compgen -W "--user --file --replace" -- "$cur") ) ;;
                    verify-backup) COMPREPLY=( $(compgen -W "--file" -- "$cur") ) ;;
                    assign-template) COMPREPLY=( $(compgen -W "--user --template" -- "$cur") ) ;;
                    assign-class) COMPREPLY=( $(compgen -W "--user --class" -- "$cur") ) ;;
                esac
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

complete -F _linuxus_completion linuxusctl ./linuxusctl
