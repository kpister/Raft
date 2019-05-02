package main

func getState(client *TYPE) {
    // Get state from each client in list
    // Store leader ID for commands (e.g. ISOLATE LEADER)
    // Store parts of responses for ASSERTS
}

func ASSERT(no bool, statement *TYPE) {
}

func leaderFunctionality(state_args...) {
    // ASSERT leader exists
    // ASSERT leader can update log
    // -- The leader's no-op has been recorded by a majority of followers
}

func clientFunctionality(state_args...) {
    // create client
    // ASSERT client.Put success
    // -- allow retries or wait time
    // ASSERT client.Get success
    // ASSERT client.Put second value for same key
    // ASSERT client.Get receives the new value
}

func logConsistency(state_args...) {
    // ASSERT that ALL persistent logs are identical
    // -- allow for empty slots
    // ASSERT that ALL logs are identical up to CommitIndex
    // -- allow for empty slots
}
