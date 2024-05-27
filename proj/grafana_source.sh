#!/bin/bash

GRAFANA_URL="http://localhost:3000"
GRAFANA_USER="admin"
GRAFANA_PASSWORD="admin"

ELASTICSEARCH_URL="http://elastic:9200"
DATASOURCE_NAME="Elasticsearch"
INDEX_NAME="containers"
TIME_FIELD_NAME="timestamp"

# Create the data source JSON payload
read -r -d '' DATA_SOURCE_JSON << EOM
{
    "name": "$DATASOURCE_NAME",
    "type": "elasticsearch",
    "access": "proxy",
    "url": "$ELASTICSEARCH_URL",
    "database": "$INDEX_NAME",
    "jsonData": {
        "timeField": "$TIME_FIELD_NAME"
    }
}
EOM

# Send a POST request to Grafana API to create the data source
RESPONSE=$(curl -s -X POST "$GRAFANA_URL/api/datasources" \
    -H "Content-Type: application/json" \
    -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
    -d "$DATA_SOURCE_JSON")

if echo "$RESPONSE" | grep -q '"id":'; then
    echo "Data source created successfully."
else
    echo "Failed to create data source: $RESPONSE"
fi
