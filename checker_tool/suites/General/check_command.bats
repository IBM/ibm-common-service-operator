@test "General | Check OC " {
    [[ "$(command -v oc 2> /dev/null)" ]]
}

@test "General | Check YQ " {
    [[ "$(command -v yq 2> /dev/null)" ]]
}