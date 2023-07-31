@test "General | OC command" {
    user=$($OC whoami 2> /dev/null)
    [[ $? -ne 0 ]]
}
