#!/usr/bin/env bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#
# This is an internal component, bundled with an official IBM product. 
# Please refer to that particular license for additional information. 

export check_cpfs_workdir=$(dirname "$(readlink -f "$BASH_SOURCE")")
export WORKTMP="/tmp"
# Where groups of test cases can be found
CHECKS_ROOT=${CHECKS_ROOT:-${check_cpfs_workdir}/suites}

function main() {
    echo "This is the main function"
    parse_arguments "$@"
}

function parse_arguments() {
    # process options
    if [[ "$@" != "" ]]; then
        case "$1" in
        --all | -a)
            run_checks 'all'
            ;;
        --groups | -g)
            run_checks 'groups' $2
            ;;
        --cases | -c)
            run_checks 'cases' $2
            ;;
        --ignore_groups | -ng)
            run_checks 'ignore_groups' $2
            ;;
        --ignore_cases | -nc)
            run_checks 'ignore_cases' $2
            ;;
        --help | -h)
            print_usage
            exit 0
            ;;
        *)
            print_usage
            exit 1
            ;;
        esac
    fi
}

function print_usage() {
    script_name=`basename ${0}`
    echo "Usage: ${script_name} --license-accept [OPTIONS]..."
    echo ""
    echo "Accepted cli arguments are:"
    echo -e "\t[--help|-h], prints this help."
    echo -e "\t[--all|-a], all cases will be ran."
    echo -e "\t[--groups|-g <groups>], the case groups should be like --groups='group1,group2'."
    echo -e "\t[--groups|-g -l], list the current supported groups."
    echo -e "\t[--ignore_groups|-ng <groups>], skip the cases groups, like -ng group1,group2."
#    echo -e "\t[--cases|-c <cases>], the cases should be like --cases='case1,case2'."
#    echo -e "\t[--cases|-c -l], list the current supported cases."
#    echo -e "\t[--ignore_cases|-nc <cases>], skip the cases groups, like -nc case1,case2."
    echo ""
}

function run_checks() {
    if [[ $1 == 'all' ]]; then
        test_suites=$(ls ${CHECKS_ROOT})
        for group in $test_suites
        do 
            test_groups="$group $test_groups"
        done
    fi

    if [[ $1 == 'groups' ]]; then
        if [[ $2 == '-l' ]]; then
            echo "The supported groups are: "
            ls -1 ./suites
            exit 0
        else
            data=$2
            temp_data=`echo ${data//,/|}`
            for group in $temp_data
            do 
                test_groups="$group $test_groups"
            done
        fi
    fi

    if [[ $1 == 'ignore_groups' ]]; then
        data=$2
        temp_data=`echo ${data//,/|}`
        test_suites=$(ls ${CHECKS_ROOT} | egrep -v "$temp_data")
        not_run_groups=$temp_data
    fi

    for group in ${test_groups[*]}; do
        for test in $(ls ${CHECKS_ROOT}/${group}/*.bats); do
            if [[ ! " ${ignore_cases[@]} " =~ " ${test} " ]]; then
                bats_files+=( ${test} )
            fi
        done
    done



    # Create the final command depending on the logging parameters
    command='bats --timing'
    command+=' ${bats_files[*]}'
    eval $command
}


main "$@"
