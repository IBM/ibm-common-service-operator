#!/usr/bin/env bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#
# This is an internal component, bundled with an official IBM product.
# Please refer to that particular license for additional information.

# ---------- Command arguments ----------

set -o errtrace
set -o errexit

CLEANUP="false"
STORAGE_CLASS="default"
ZEN="false"
ZEN4="false"
IM="false"
KEYCLOAK="false"
MONGO="false"
UTIL="false"
SELECTED="false"

function main() {
  parse_arguments "$@"
  if [[ $CLEANUP == "true" ]]; then
    cleanup
  else
    deploy_resources
  fi
}

function parse_arguments() {
  # process options
  while [[ "$@" != "" ]]; do
    case "$1" in
    --services-ns)
      shift
      TARGET_NAMESPACE=$1
      ;;
    --operator-ns)
      shift
      OPERATOR_NAMESPACE=$1
      ;;
    --tethered-ns)
      shift
      TETHERED_NS=$1
      ;;
    --zen)
      ZEN="true"
      SELECTED="true"
      ;;
    --zen4)
      ZEN4="true"
      SELECTED="true"
      ;;
    --im)
      IM="true"
      SELECTED="true"
      ;;
    --keycloak)
      KEYCLOAK="true"
      SELECTED="true"
      ;;
    --mongo)
      MONGO="true"
      SELECTED="true"
      ;;
    --util)
      UTIL="true"
      SELECTED="true"
      ;;
    --storage-class)
      shift
      STORAGE_CLASS=$1
      ;;
    -c | --cleanup)
      CLEANUP="true"
      ;;
    -h | --help)
      print_usage
      exit 1
      ;;
    *) 
      warning "$1 not a supported parameter for deploy-br-resoruces.sh"
      ;;
    esac
    shift
  done
  if [[ $SELECTED == "false" ]]; then
    error "No component selected. Please use a combination of --im, --mongo, --keycloak, --zen, or --zen4 to select components to deploy resources for."
  fi
  if [[ $TARGET_NAMESPACE == "" ]] && [[ $OPERATOR_NAMESPACE == "" ]]; then
    error "No namespace selected. Please re-run script with --services-ns and/or --operator-ns parameter defined."
  fi
  if [[ $UTIL == "true" ]] && [[ $OPERATOR_NAMESPACE == "" ]]; then
    error "CPFS Util selected but no operator namespace provided. Please re-run script with --operator-ns parameter defined."
  fi
  if [[ $ZEN == "true" && $ZEN4 == "true" ]]; then
    error "Cannot select --zen and --zen4 on the same run of the script. Please verify zen version in the namespace $TARGET_NAMESPACE and select the appropriate option."
  fi
  if [[ $OPERATOR_NAMESPACE != "" ]] && [[ $TARGET_NAMESPACE == "" ]]; then
    warning "No services namespace specified, using operator namespace $OPERATOR_NAMESPACE instead. If using SOD topology, please re-run with both --operator-ns and --services-ns defined."
    TARGET_NAMESPACE=$OPERATOR_NAMESPACE
  fi
}

function print_usage() {
  echo "Usage: ${script_name} --<service> --operator-ns --services-ns [OPTIONS]..."
  echo ""
  echo "Deploy the necessary resources for Backup of Keycloak."
  #TODO change below to point to correct docs
  #echo "See step 4 here https://www.ibm.com/docs/en/cloud-paks/foundational-services/4.0?topic=4x-isolated-migration for more information."
  echo ""
  echo "Options:"
  echo "   --operator-ns string                           Optional. Operator namespace for a given CPFS installation. Only required if --util specified"
  echo "   --services-ns string                           Required. Namespace where IM EDB, IM Mongo, Zen, or Keycloak operands are installed. If installed in different namespaces, script will need to be run separately. Optional if it is the same as --operator-ns."
  echo "   --tethered-ns string                           Optional. Comma-delimited list of namespaces attached to a given CPFS install. Should include any namespace from a given tenant in the common-service-maps cm that are not the --operator-ns or the --services-ns. Only required when --util specified."
  echo "   --im, --mongo, --keycloak, --zen, --zen4       Required. Choose which component(s) to deploy backup/restore resources for to the target namespace. At least one is required. Multiple can be specified but only one of --zen or --zen4 can be chosen."
  echo "   --util                                         Optional. Deploy the CPFS Util job for use in SOD topologies. If this option is selected, --operator-ns and --services-ns are both required."
  echo "   --storage-class string                         Optional. Storage class to use for backup/restore resources. Default value is cluster's default storage class."
  echo "   -c, --cleanup                                  Optional. Automated cleanup of backup/restore resources. Will run cleanup instead of deployment logic."
  echo "   -h, --help                                     Print usage information"
  echo ""
}

function deploy_resources(){
  if [[ $STORAGE_CLASS == "default" ]]; then
    STORAGE_CLASS=$(oc get sc | grep default | awk '{print $1}')
    info "Using default storage class $STORAGE_CLASS."
  else
    info "Using specified storage class $STORAGE_CLASS."
  fi

  #deploy IM EDB resources
  if [[ $IM == "true" ]]; then
    info "Creating IM Backup/Restore resources in namespace $TARGET_NAMESPACE."
    sed -i -E "s/<cs-db namespace>/$TARGET_NAMESPACE/" common-service-db/cs-db-backup-deployment.yaml
    sed -i -E "s/<cs-db namespace>/$TARGET_NAMESPACE/" common-service-db/cs-db-backup-pvc.yaml
    sed -i -E "s/<storage class>/$STORAGE_CLASS/" common-service-db/cs-db-backup-pvc.yaml
    sed -i -E "s/<cs-db namespace>/$TARGET_NAMESPACE/" common-service-db/cs-db-role.yaml
    sed -i -E "s/<cs-db namespace>/$TARGET_NAMESPACE/" common-service-db/cs-db-rolebinding.yaml
    sed -i -E "s/<cs-db namespace>/$TARGET_NAMESPACE/" common-service-db/cs-db-sa.yaml
    sed -i -E "s/<cs-db namespace>/$TARGET_NAMESPACE/" common-service-db/cs-db-br-script-cm.yaml
    oc apply -f ./common-service-db -n $TARGET_NAMESPACE || error "Unable to deploy resources for IM."
    success "Resources to backup IM deployed in namespace $TARGET_NAMESPACE."
  fi

  #Deploy IM Mongo resources
  if [[ $MONGO == "true" ]]; then
    info "Creating IM Mongo Backup/Restore resources in namespace $TARGET_NAMESPACE."
    sed -i -E "s~<mongo namespace>~$TARGET_NAMESPACE~g" mongodb-backup-deployment.yaml
    sed -i -E "s/<mongo namespace>/$TARGET_NAMESPACE/" mongodb-backup-pvc.yaml
    sed -i -E "s/<storage class>/$STORAGE_CLASS/" mongodb-backup-pvc.yaml
    oc apply -f mongodb-backup-deployment.yaml -f mongodb-backup-pvc.yaml -n $TARGET_NAMESPACE || error "Unable to deploy resources for IM Mongo."
    success "Resources to backup IM Mongo deployed in namespace $TARGET_NAMESPACE."
  fi

  #Deploy Keycloak resources
  if [[ $KEYCLOAK == "true" ]]; then 
    info "Creating Keycloak Backup/Restore resources in namespace $TARGET_NAMESPACE."
    sed -i -E "s/<keycloak namespace>/$TARGET_NAMESPACE/" keycloak/keycloak-backup-deployment.yaml
    sed -i -E "s/<keycloak namespace>/$TARGET_NAMESPACE/" keycloak/keycloak-backup-pvc.yaml
    sed -i -E "s/<storage class>/$STORAGE_CLASS/" keycloak/keycloak-backup-pvc.yaml
    sed -i -E "s/<keycloak namespace>/$TARGET_NAMESPACE/" keycloak/keycloak-role.yaml
    sed -i -E "s/<keycloak namespace>/$TARGET_NAMESPACE/" keycloak/keycloak-rolebinding.yaml
    sed -i -E "s/<keycloak namespace>/$TARGET_NAMESPACE/" keycloak/keycloak-sa.yaml
    sed -i -E "s/<keycloak namespace>/$TARGET_NAMESPACE/" keycloak/keycloak-br-script-cm.yaml
    oc apply -f ./keycloak -n $TARGET_NAMESPACE || error "Unable to deploy resources for Keycloak."
    success "Resources to backup Keycloak deployed in namespace $TARGET_NAMESPACE."
  fi

  #Deploy zen 5 resources
  if [[ $ZEN == "true" ]]; then
    if [[ $ZENSERVICE == "" ]]; then
      ZENSERVICE=$(oc get zenservice -n $TARGET_NAMESPACE --no-headers | awk '{print $1}')
    fi
    if [[ $ZENSERVICE != "" ]]; then
      exists=$(oc get zenservice $ZENSERVICE -n $TARGET_NAMESPACE --no-headers --ignore-not-found)
      if [[ $exists == "" ]]; then
        error "Zenservice $ZENSERVICE not found in namespace $TARGET_NAMESPACE. Make sure the zenservice is deployed to the target namespace $TARGET_NAMESPACE or change the namespace used."
      else
        info "Creating Zen Backup/Restore resources in namespace $TARGET_NAMESPACE."
        sed -i -E "s/<zenservice namespace>/$TARGET_NAMESPACE/" zen5-backup-deployment.yaml
        sed -i -E "s/<zenservice name>/$ZENSERVICE/" zen5-backup-deployment.yaml
        sed -i -E "s/<zenservice namespace>/$TARGET_NAMESPACE/" zen5-backup-pvc.yaml
        sed -i -E "s/<storage class>/$STORAGE_CLASS/" zen5-backup-pvc.yaml
        sed -i -E "s/<zenservice namespace>/$TARGET_NAMESPACE/" zen5-role.yaml
        sed -i -E "s/<zenservice namespace>/$TARGET_NAMESPACE/" zen5-rolebinding.yaml
        sed -i -E "s/<zenservice namespace>/$TARGET_NAMESPACE/" zen5-sa.yaml
        sed -i -E "s/<zenservice namespace>/$TARGET_NAMESPACE/" zen5-br-scripts-cm.yaml
        oc apply -f zen5-backup-deployment.yaml -f zen5-backup-pvc.yaml -f zen5-role.yaml -f zen5-rolebinding.yaml -f zen5-sa.yaml -f zen5-br-scripts-cm.yaml -n $TARGET_NAMESPACE || error "Unable to deploy resources for Zen 5."      
      fi
    fi
    success "Resources to backup Zen deployed in namespace $TARGET_NAMESPACE."
  fi

  #deploy zen 4 resources
  if [[ $ZEN4 == "true" ]]; then
    if [[ $ZENSERVICE == "" ]]; then
      ZENSERVICE=$(oc get zenservice -n $TARGET_NAMESPACE --no-headers | awk '{print $1}')
    fi
    if [[ $ZENSERVICE != "" ]]; then
      exists=$(oc get zenservice $ZENSERVICE -n $TARGET_NAMESPACE --no-headers --ignore-not-found)
      if [[ $exists == "" ]]; then
        error "Zenservice $ZENSERVICE not found in namespace $TARGET_NAMESPACE. Make sure the zenservice is deployed to the target namespace $TARGET_NAMESPACE or change the namespace used."
      else
        info "Creating Zen Backup/Restore resources in namespace $TARGET_NAMESPACE."
        sed -i -E "s/<zenservice namespace>/$TARGET_NAMESPACE/" zen-backup-deployment.yaml
        sed -i -E "s/<zenservice namespace>/$TARGET_NAMESPACE/" zen-backup-pvc.yaml
        sed -i -E "s/<storage class>/$STORAGE_CLASS/" zen-backup-pvc.yaml
        sed -i -E "s/<zenservice namespace>/$TARGET_NAMESPACE/" zen4-role.yaml
        sed -i -E "s/<zenservice namespace>/$TARGET_NAMESPACE/" zen4-rolebinding.yaml
        sed -i -E "s/<zenservice namespace>/$TARGET_NAMESPACE/" zen4-sa.yaml
        sed -i -E "s/<zenservice namespace>/$TARGET_NAMESPACE/" zen4-br-scripts.yaml
        oc apply -f zen4-backup-deployment.yaml -f zen4-backup-pvc.yaml -f zen4-role.yaml -f zen4-rolebinding.yaml -f zen4-sa.yaml -f zen4-br-scripts-cm.yaml -n $TARGET_NAMESPACE || error "Unable to deploy resources for Zen 4."
      fi        
    fi
    success "Resources to backup Zen 4 deployed in namespace $TARGET_NAMESPACE."
  fi
  
  success "Backup/Restore resources created in namespace $TARGET_NAMESPACE."

  #Deploy cpfs-util resources
  if [[ $UTIL == "true" ]]; then
    info "Creating CPFS Util Backup/Restore resources in namespace $OPERATOR_NAMESPACE."
    sed -i -E "s/<operator namespace>/$OPERATOR_NAMESPACE/" ../spectrum-fusion/cpfs-util-resources/cpfs-util-job.yaml
    sed -i -E "s/<operator namespace>/$OPERATOR_NAMESPACE/" ../spectrum-fusion/cpfs-util-resources/cpfs-util-pvc.yaml
    sed -i -E "s/<storage class>/$STORAGE_CLASS/" ../spectrum-fusion/cpfs-util-resources/cpfs-util-pvc.yaml
    sed -i -E "s/<operator namespace>/$OPERATOR_NAMESPACE/" ../spectrum-fusion/cpfs-util-resources/cpfs-util-role.yaml
    sed -i -E "s/<operator namespace>/$OPERATOR_NAMESPACE/" ../spectrum-fusion/cpfs-util-resources/cpfs-util-rolebinding.yaml
    sed -i -E "s/<operator namespace>/$OPERATOR_NAMESPACE/" ../spectrum-fusion/cpfs-util-resources/cpfs-util-sa.yaml
    sed -i -E "s/<operator namespace>/$OPERATOR_NAMESPACE/" ../spectrum-fusion/cpfs-util-resources/cpfs-util-br-configmap.yaml
    sed -i -E "s/<services or tethered namespace>/$TARGET_NAMESPACE/" ../spectrum-fusion/cpfs-util-resources/cpfs-util-serv-tethered-role.yaml
    sed -i -E "s/<services or tethered namespace>/$TARGET_NAMESPACE/" ../spectrum-fusion/cpfs-util-resources/cpfs-util-serv-tethered-rolebinding.yaml
    if [[ $TETHERED_NS != "" ]]; then
      for ns in ${TETHERED_NS//,/ }; do
        sed -i -E "s/<services or tethered namespace>/$ns/" ../spectrum-fusion/cpfs-util-resources/cpfs-util-serv-tethered-role.yaml
        sed -i -E "s/<services or tethered namespace>/$ns/" ../spectrum-fusion/cpfs-util-resources/cpfs-util-serv-tethered-rolebinding.yaml
      done
    fi
    oc apply -f ../spectrum-fusion/cpfs-util-resources -n $OPERATOR_NAMESPACE || error "Unable to deploy resources for CPFS Util."
    success "CPFS Util resources deployed in namespace $OPERATOR_NAMESPACE."
  fi

}

function cleanup() {
  title "Cleaning Backup/Restore resources deployed to namespace $TARGET_NAMESPACE."
  #clean up IM EDB BR resources
  if [[ $IM == "true" ]]; then
    info "Clean up IM BR resources..."
    oc delete deploy cs-db-backup -n $TARGET_NAMESPACE --ignore-not-found && oc delete sa cs-db-backup-sa -n $TARGET_NAMESPACE --ignore-not-found && oc delete role cs-db-backup-role -n $TARGET_NAMESPACE --ignore-not-found && oc delete rolebinding cs-db-backup-rolebinding -n $TARGET_NAMESPACE --ignore-not-found
    pod=$(oc get pod -n $TARGET_NAMESPACE --no-headers --ignore-not-found | grep cs-db-backup | awk '{print $1}' | tr "\n" " ")
    if [[ $pod != "" ]]; then
      oc delete pod $pod -n $TARGET_NAMESPACE --ignore-not-found || warning "IM backup pod not found, moving on."
    fi
    oc delete pvc cs-db-backup-pvc -n $TARGET_NAMESPACE --ignore-not-found
    oc delete cm cs-db-br-configmap -n $TARGET_NAMESPACE --ignore-not-found
    success "IM BR resources cleaned up."
  fi

  #Clean up IM Mongo BR resources
  if [[ $MONGO == "true" ]]; then
    info "Clean up IM Mongo BR resources..."
    oc delete deploy mongodb-backup -n $TARGET_NAMESPACE --ignore-not-found && oc delete sa cs-db-backup-sa -n $TARGET_NAMESPACE --ignore-not-found && oc delete role cs-db-backup-role -n $TARGET_NAMESPACE --ignore-not-found && oc delete rolebinding cs-db-backup-rolebinding -n $TARGET_NAMESPACE --ignore-not-found
    pod=$(oc get pod -n $TARGET_NAMESPACE --no-headers --ignore-not-found | grep cs-db-backup | awk '{print $1}' | tr "\n" " ")
    if [[ $pod != "" ]]; then
      oc delete pod $pod -n $TARGET_NAMESPACE --ignore-not-found || warning "IM backup pod not found, moving on."
    fi
    oc delete pvc cs-mongodump -n $TARGET_NAMESPACE --ignore-not-found
    success "IM BR resources cleaned up."
  fi

  #Clean up Keycloak BR resources
  if [[ $KEYCLOAK == "true" ]]; then
    info "Clean up Keycloak BR resources..."
    oc delete deploy keycloak-backup -n $TARGET_NAMESPACE --ignore-not-found && oc delete sa keycloak-backup-sa -n $TARGET_NAMESPACE --ignore-not-found && oc delete role keycloak-backup-role -n $TARGET_NAMESPACE --ignore-not-found && oc delete rolebinding keycloak-backup-rolebinding -n $TARGET_NAMESPACE --ignore-not-found
    pod=$(oc get pod -n $TARGET_NAMESPACE --no-headers --ignore-not-found | grep keycloak-backup | awk '{print $1}' | tr "\n" " ")
    if [[ $pod != "" ]]; then
      oc delete pod $pod -n $TARGET_NAMESPACE --ignore-not-found || warning "Keycloak backup pod not found, moving on."
    fi
    oc delete pvc keycloak-backup-pvc -n $TARGET_NAMESPACE --ignore-not-found
    oc delete cm keycloak-br-configmap -n $TARGET_NAMESPACE --ignore-not-found
    success "IM Mongo BR resources cleaned up."
  fi

  #Clean up zen 5 BR resources
  if [[ $ZEN == "true" ]]; then
    info "Clean up Zen BR resources..."
    oc delete deploy zen5-backup -n $TARGET_NAMESPACE --ignore-not-found && oc delete sa zen5-backup-sa -n $TARGET_NAMESPACE --ignore-not-found && oc delete role zen5-backup-role -n $TARGET_NAMESPACE --ignore-not-found && oc delete rolebinding zen5-backup-rolebinding -n $TARGET_NAMESPACE --ignore-not-found
    pod=$(oc get pod -n $TARGET_NAMESPACE --no-headers --ignore-not-found | grep zen5-backup | awk '{print $1}' | tr "\n" " ")
    if [[ $pod != "" ]]; then
      oc delete pod $pod -n $TARGET_NAMESPACE --ignore-not-found || warning "Zen backup pod not found, moving on."
    fi
    oc delete pvc zen5-backup-pvc -n $TARGET_NAMESPACE --ignore-not-found
    oc delete cm zen5-br-configmap -n $TARGET_NAMESPACE --ignore-not-found
    success "Zen BR resources cleaned up."
  fi

  #clean up zen 4 BR resources
  if [[ $ZEN4 == "true" ]]; then
    info "Clean up Zen 4 BR resources..."
    oc delete deploy zen4-backup -n $TARGET_NAMESPACE --ignore-not-found && oc delete sa zen4-backup-sa -n $TARGET_NAMESPACE --ignore-not-found && oc delete role zen4-backup-role -n $TARGET_NAMESPACE --ignore-not-found && oc delete rolebinding zen4-backup-rolebinding -n $TARGET_NAMESPACE --ignore-not-found
    pod=$(oc get pod -n $TARGET_NAMESPACE --no-headers --ignore-not-found | grep zen4-backup | awk '{print $1}' | tr "\n" " ")
    if [[ $pod != "" ]]; then
      oc delete pod $pod -n $TARGET_NAMESPACE --ignore-not-found || warning "Zen 4 backup pod not found, moving on."
    fi
    oc delete pvc zen4-backup-pvc -n $TARGET_NAMESPACE --ignore-not-found
    oc delete cm zen4-br-configmap -n $TARGET_NAMESPACE --ignore-not-found
    success "Zen 4 BR resources cleaned up."
  fi
  
  success "BR resources succesfully removed from namespace $TARGET_NAMESPACE."
}

function msg() {
  printf '%b\n' "$1"
}

function success() {
  msg "\33[32m[✔] ${1}\33[0m"
}

function warning() {
  msg "\33[33m[✗] ${1}\33[0m"
}

function error() {
  msg "\33[31m[✘] ${1}\33[0m"
  exit 1
}

function title() {
  msg "\33[34m# ${1}\33[0m"
}

function info() {
  msg "[INFO] ${1}"
}

main $*