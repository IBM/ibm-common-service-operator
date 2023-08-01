function check_command() {
    local command=$1

    if [[ -z "$(command -v ${command} 2> /dev/null)" ]]; then
        error "${command} command not available"
    else
        success "${command} command available"
    fi
}

function success() {
  msg "\33[32m[✔] ${1}\33[0m"
}

function error() {
  msg "\33[31m[✘] ${1}\33[0m"
}

function msg() {
    printf '%b\n' "$1"
}


check_command "oc"
check_command "yq"

