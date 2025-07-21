#!/bin/bash
# pg_dump wrapper that automatically selects the correct version

# Function to extract major version from PostgreSQL version string
get_major_version() {
    local version_string="$1"
    # Extract major version number (e.g., "16" from "PostgreSQL 16.2")
    echo "$version_string" | grep -oE 'PostgreSQL [0-9]+' | grep -oE '[0-9]+$'
}

# Function to find the best matching pg_dump version
find_best_pg_dump() {
    local server_version="$1"
    local available_versions=(17 16 15)
    
    # For older versions, use pg_dump15 (backward compatible)
    if [ "$server_version" -lt 15 ]; then
        server_version=15
    fi
    
    # First, try exact match
    for v in "${available_versions[@]}"; do
        if [ "$v" -eq "$server_version" ] && [ -x "/usr/bin/pg_dump$v" ]; then
            echo "/usr/bin/pg_dump$v"
            return 0
        fi
    done
    
    # If no exact match, use the closest version that's >= server version
    for v in "${available_versions[@]}"; do
        if [ "$v" -ge "$server_version" ] && [ -x "/usr/bin/pg_dump$v" ]; then
            echo "/usr/bin/pg_dump$v"
            return 0
        fi
    done
    
    # Fallback to the newest available version
    for v in "${available_versions[@]}"; do
        if [ -x "/usr/bin/pg_dump$v" ]; then
            echo "/usr/bin/pg_dump$v"
            return 0
        fi
    done
    
    return 1
}

# Check if we have a DATABASE_URL or connection parameters
DATABASE_URL=""
for arg in "$@"; do
    # Check if this argument is the database URL (last non-option argument)
    if [[ ! "$arg" =~ ^- ]]; then
        DATABASE_URL="$arg"
    fi
done

# If no DATABASE_URL in args, check environment
if [ -z "$DATABASE_URL" ]; then
    DATABASE_URL="${DATABASE_URL:-}"
fi

if [ -z "$DATABASE_URL" ]; then
    echo "Error: No database connection found" >&2
    exit 1
fi

# Get PostgreSQL server version
VERSION_OUTPUT=$(psql "$DATABASE_URL" -t -c "SELECT version();" 2>/dev/null)
if [ $? -ne 0 ]; then
    echo "Warning: Could not determine PostgreSQL version, using latest pg_dump" >&2
    # Default to the latest version
    exec /usr/bin/pg_dump17 "$@"
fi

# Extract major version
SERVER_VERSION=$(get_major_version "$VERSION_OUTPUT")
if [ -z "$SERVER_VERSION" ]; then
    echo "Warning: Could not parse PostgreSQL version from: $VERSION_OUTPUT" >&2
    exec /usr/bin/pg_dump17 "$@"
fi

# Find the best pg_dump version
PG_DUMP_BIN=$(find_best_pg_dump "$SERVER_VERSION")
if [ $? -ne 0 ]; then
    echo "Error: No suitable pg_dump version found" >&2
    exit 1
fi

echo "Using $PG_DUMP_BIN for PostgreSQL $SERVER_VERSION" >&2

# Execute the selected pg_dump with all original arguments
exec "$PG_DUMP_BIN" "$@"