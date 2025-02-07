#!/usr/bin/env bash
#goals
# 1 delete operator (sub, csv)
    # nss, odlm, cs op
# 2 delete rbac
# 3 delete any other resources that will be replaced by helm install (be mindful of CRs that need to be running like auth)
# helm install (not part of this script)

#this is a test script as a proof of concept

#start with CS operator, if we start with odlm cs operator will recreate it
#may need to delete the webhook and service (keep an eye out to see if odlm deletes it)
operatorNamespace=$1
servicesNamespace=$2

nssSub=$(oc get subscription.operators.coreos.com -n "$operatorNamespace" -o jsonpath="{.items[?(@.spec.name=='ibm-namespace-scope-operator')].metadata.name}")
nssCSV=$(oc get --ignore-not-found sub $nssSub -n $operatorNamespace -o jsonpath='{.status.currentCSV}')
nssVersion=$(oc get --ignore-not-found csv $nssCSV -n $operatorNamespace -o jsonpath='{.spec.version}')

csSub=$(oc get subscription.operators.coreos.com -n "$operatorNamespace" -o jsonpath="{.items[?(@.spec.name=='ibm-common-service-operator')].metadata.name}")
csCSV=$(oc get --ignore-not-found sub $csSub -n $operatorNamespace -o jsonpath='{.status.currentCSV}')
csVersion=$(oc get --ignore-not-found csv $csCSV -n $operatorNamespace -o jsonpath='{.spec.version}')

odlmSub=$(oc get subscription.operators.coreos.com -n "$operatorNamespace" -o jsonpath="{.items[?(@.spec.name=='ibm-odlm')].metadata.name}")
odlmCSV=$(oc get --ignore-not-found sub $odlmSub -n $operatorNamespace -o jsonpath='{.status.currentCSV}')
odlmVersion=$(oc get --ignore-not-found csv $odlmCSV -n $operatorNamespace -o jsonpath='{.spec.version}')

iamSub=$(oc get subscription.operators.coreos.com -n "$operatorNamespace" -o jsonpath="{.items[?(@.spec.name=='ibm-iam-operator')].metadata.name}")
iamCSV=$(oc get --ignore-not-found sub $iamSub -n $operatorNamespace -o jsonpath='{.status.currentCSV}')

uiSub=$(oc get subscription.operators.coreos.com -n "$operatorNamespace" -o jsonpath="{.items[?(@.spec.name=='ibm-commonui-operator-app')].metadata.name}")
uiCSV=$(oc get --ignore-not-found sub $uiSub -n $operatorNamespace -o jsonpath='{.status.currentCSV}')

oc delete --ignore-not-found csv $csCSV -n $operatorNamespace && oc delete --ignore-not-found sub $csSub -n $operatorNamespace
oc delete --ignore-not-found csv $nssCSV -n $operatorNamespace && oc delete --ignore-not-found sub $nssSub -n $operatorNamespace
#helm conditional for im
oc delete --ignore-not-found csv $iamCSV -n $operatorNamespace && oc delete --ignore-not-found sub $iamSub -n $operatorNamespace
oc delete --ignore-not-found csv $uiCSV -n $operatorNamespace && oc delete --ignore-not-found sub $uiSub -n $operatorNamespace
#end helm conditional
sleep 30
oc delete --ignore-not-found csv $odlmCSV -n $operatorNamespace && oc delete --ignore-not-found sub $odlmSub -n $operatorNamespace
oc delete --ignore-not-found role operand-deployment-lifecycle-manager.$odlmVersion -n $operatorNamespace && oc delete --ignore-not-found rolebinding operand-deployment-lifecycle-manager.$odlmVersion -n $operatorNamespace

#loop for removing roles from services and tethered namespace
namespaces=$(oc get cm namespace-scope -n $operatorNamespace -o jsonpath="{.data.namespaces}")
for ns in ${namespaces//,/}; do
    roles=""
    #get cs operator roles
    $roles="${roles} $(oc get roles -n $ns | grep ibm-common-service-op | awk '{print $1}' | tr "\n" " ")"
    #get odlm roles
    $roles="${roles} $(oc get roles -n $ns | grep operand-deployment-l | awk '{print $1}' | tr "\n" " ")"
    #get iam roles
    $roles="${roles} $(oc get roles -n $ns | grep ibm-iam | awk '{print $1}' | tr "\n" " ")"
    #get ui roles
    $roles="${roles} $(oc get roles -n $ns | grep ibm-commonui | awk '{print $1}' | tr "\n" " ")"
    #get nss roles?
    echo "${roles}"

    oc delete role $roles -n $ns --ignore-not-found
done

#probably need a loop for all namespaces installer has access to
#can get list of namespaces from nss cm data field, then iterate based on comma delimited result

#ODLM's csv creates two operand requests, the second one has a hash value attached to it reading operand-deployment-lifec-3hAJOqfA3NkKQ6J3Rr9xQB2SeWbQRtvLDEDzS1 
# in the services namespace, the role name reads operand-deployment-lifec-3hAJOqfA3NkKQ6J3Rr9xQB2SeWbQRtvLDEDzS1-dc353b20e344fd
# I am not sure if the second hash value is the same in all tenant namespaces, may have to iterate across tenant namespaces
# in that case might make more sense to iterate first and do everything in the same loop since every operator is setup this way
# for now stick to operator and services for PoC

# oc delete --ignore-not-found role ibm-namespace-scope-operator.$nssVersion -n $operatorNamespace && oc delete --ignore-not-found rolebinding ibm-namespace-scope-operator.$nssVersion -n $operatorNamespace
# oc delete --ignore-not-found role ibm-common-service-operator.$csVersion -n $operatorNamespace && oc delete --ignore-not-found rolebinding ibm-common-service-operator.$csVersion -n $operatorNamespace
# oc delete --ignore-not-found csv $odlmCSV -n $operatorNamespace && oc delete --ignore-not-found sub $odlmSub -n $operatorNamespace
# oc delete --ignore-not-found role operand-deployment-lifecycle-manager.$odlmVersion -n $operatorNamespace && oc delete --ignore-not-found rolebinding operand-deployment-lifecycle-manager.$odlmVersion -n $operatorNamespace
# remainingRole=$(oc get --ignore-not-found role -n $operatorNamespace | grep operand-deployment | awk '{print $1}')
# oc delete --ignore-not-found role $remainingRole -n $operatorNamespace && oc delete --ignore-not-found rolebinding $remainingRole -n $operatorNamespace
# servicesRole=$(oc get --ignore-not-found role -n $servicesNamespace | grep $remainingRole | awk '{print $1}')
# oc delete --ignore-not-found role $servicesRole -n $servicesNamespace && oc delete --ignore-not-found rolebinding $servicesRole -n $servicesNamespace
# oc delete --ignore-not-found serviceaccount operand-deployment-lifecycle-manager -n $operatorNamespace
# oc delete --ignore-not-found deploy operand-deployment-lifecycle-manager -n $operatorNamespace