#!/bin/bash

GRAFANA_URL="http://localhost:3000"
GRAFANA_USER="admin"
GRAFANA_PASSWORD="admin"

# Get a list of all data sources
DATA_SOURCES=$(curl -s -X GET "$GRAFANA_URL/api/datasources" \
    -u "$GRAFANA_USER:$GRAFANA_PASSWORD")

# Parse the data source IDs from the JSON response
DATA_SOURCE_IDS=$(echo "$DATA_SOURCES" | jq -r '.[].id')

# Iterate over each data source ID and delete the data source
for ID in $DATA_SOURCE_IDS; do
    RESPONSE=$(curl -s -X DELETE "$GRAFANA_URL/api/datasources/$ID" \
        -u "$GRAFANA_USER:$GRAFANA_PASSWORD")

    if [ "$RESPONSE" == "" ]; then
        echo "Data source with ID $ID deleted successfully."
    else
        echo "Failed to delete data source with ID $ID: $RESPONSE"
    fi
done
