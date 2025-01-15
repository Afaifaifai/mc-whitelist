#!/bin/bash

# Check if a parameter is provided
if [ -z "$1" ]; then
    echo "Usage: $0 <player_name>"
    exit 1
fi

# Receive the parameter
PLAYER_NAME=$1

# Execute the Docker command
docker exec mc rcon-cli whitelist add "$PLAYER_NAME"

# Confirm execution status
if [ $? -eq 0 ]; then
    echo "Successfully added $PLAYER_NAME to the whitelist."
else
    echo "Execution failed. Please check Docker and RCON settings."
fi