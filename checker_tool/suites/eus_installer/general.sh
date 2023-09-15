function check_oc_login() {
    # Checking oc command logged in
    user=$(oc whoami 2> /dev/null)
    if [ $? -ne 0 ]; then
        error "You must be logged into the OpenShift Cluster from the oc command line"
    else
        success "oc command logged in as ${user}"
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

check_oc_login