#!/bin/bash
# psql wrapper that automatically selects the correct version

# Function to extract major version from PostgreSQL version string
get_major_version() {
    local version_string="$1"
    echo "$version_string" | grep -oE 'PostgreSQL [0-9]+' | grep -oE '[0-9]+$'
}

# Function to find the best matching psql version
find_best_psql() {
    local server_version="$1"
    local available_versions=(17 16 15)
    
    # For older versions, use psql15 (backward compatible)
    if [ "$server_version" -lt 15 ]; then
        server_version=15
    fi
    
    # First, try exact match
    for v in "${available_versions[@]}"; do
        if [ "$v" -eq "$server_version" ] && [ -x "/usr/bin/psql$v" ]; then
            echo "/usr/bin/psql$v"
            return 0
        fi
    done
    
    # If no exact match, use the closest version that's >= server version
    for v in "${available_versions[@]}"; do
        if [ "$v" -ge "$server_version" ] && [ -x "/usr/bin/psql$v" ]; then
            echo "/usr/bin/psql$v"
            return 0
        fi
    done
    
    # Fallback to the newest available version
    for v in "${available_versions[@]}"; do
        if [ -x "/usr/bin/psql$v" ]; then
            echo "/usr/bin/psql$v"
            return 0
        fi
    done
    
    return 1
}

# For psql, we need to handle the case where we're just getting the version
# Check if we're being called with simple version query
for arg in "$@"; do
    if [[ "$arg" == "--version" ]] || [[ "$arg" == "-V" ]]; then
        # Just use the latest version for version queries
        exec /usr/bin/psql17 "$@"
    fi
done

# Try to detect server version from connection
# This is trickier for psql since it might be used interactively
# For now, default to the latest version and let PostgreSQL handle compatibility
PSQL_BIN="/usr/bin/psql17"

# If we can detect a DATABASE_URL, try to get the server version
DATABASE_URL=""
for arg in "$@"; do
    if [[ ! "$arg" =~ ^- ]]; then
        DATABASE_URL="$arg"
        break
    fi
done

if [ -z "$DATABASE_URL" ]; then
    DATABASE_URL="${DATABASE_URL:-}"
fi

# If we have a connection string, try to detect version
if [ -n "$DATABASE_URL" ]; then
    VERSION_OUTPUT=$(/usr/bin/psql17 "$DATABASE_URL" -t -c "SELECT version();" 2>/dev/null)
    if [ $? -eq 0 ]; then
        SERVER_VERSION=$(get_major_version "$VERSION_OUTPUT")
        if [ -n "$SERVER_VERSION" ]; then
            BEST_PSQL=$(find_best_psql "$SERVER_VERSION")
            if [ $? -eq 0 ]; then
                PSQL_BIN="$BEST_PSQL"
            fi
        fi
    fi
fi

# Execute the selected psql with all original arguments
exec "$PSQL_BIN" "$@"