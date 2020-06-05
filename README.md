# Helm to Operator migration

This branch hosts the IBM Common Services Helm(3.2.4) to Operator(3.4) migration.

There are 4 jobs used to backup a 3.2.4 version of the common services and restore to the 3.4 version.

1. Create RBAC for backup and restore jobs

    ```
    oc create -f cs-job-rbac.yaml
    ```

2. Run the cs3.2.4 init job to create a volume that will be used for backup. This job will set up a volume in OCP where data can  stored for restore after installing Common Services 3.4.  The job runs in the kube-system namespace and should be checked for successful completion

    ```
    oc create -f cs3.2.4-init.yaml

    # waiting a few seconds, then run following to check logs
    oc logs -n kube-system -f $(oc get pod -n kube-system -l job-name=cs324-init -o jsonpath="{.items[].metadata.name}")

    # after the job run finished, checking the pvc cs-backupdata status to make sure the pvc was bound;
    oc get pvc -n kube-system cs-backupdata -o jsonpath={.status.phase}
    ```

3. Run the backup job.  This will make a copy of the mongodb database and all pertinent kubernetes artifacts needed for the restore.  Check the job for successful completion.

    ```
    oc create -f cs3.2.4-backup.yaml

    # waiting a few seconds, then run following to check logs
    oc logs -n kube-system -f $(oc get pod -n kube-system -l job-name=cs324-backup -o jsonpath="{.items[].metadata.name}")
    ```

4. Uninstall Common Services 3.2.4.  This is done by running the following command to launch the uninstall job:.  Check the job for successfulk completion.

    ```
    oc create -f cs3.2.4-uninstall.yaml

    # waiting a few seconds, then run following to check logs
    oc logs -n kube-system -f $(oc get pod -n kube-system -l job-name=cs324-uninstall -o jsonpath="{.items[].metadata.name}")
    ```

5. Install Common Services 3.4 and restore information from 3.2.4.  Run the following job to do this:

    ```
    oc create -f cs3.4-restore.yaml

    # waiting a few seconds, then run following to check logs
    oc logs -n kube-system -f $(oc get pod -n kube-system -l job-name=cs34-restore -o jsonpath="{.items[].metadata.name}")
    ```

    Upon successful completion of this job, the upgrade is complete.

6. Clean up upgrade resources

    ```
    oc delete -f cs3.2.4-init.yaml
    oc delete -f cs3.2.4-backup.yaml
    oc delete -f cs3.2.4-uninstall.yaml
    oc delete -f cs3.4-restore.yaml
    oc delete -f cs-job-rbac.yaml
    ```
