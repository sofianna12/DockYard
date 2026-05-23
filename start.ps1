# On Windows with Docker Desktop the socket is world-readable inside containers,
# so DOCKER_GID=0 (root group fallback) is sufficient.
$content = Get-Content .env -Raw -ErrorAction SilentlyContinue
if ($content -match "(?m)^DOCKER_GID=") {
    $content = $content -replace "(?m)^DOCKER_GID=.*", "DOCKER_GID=0"
    Set-Content .env $content.TrimEnd()
} else {
    Add-Content .env "DOCKER_GID=0"
}
$env:DOCKER_GID = 0
docker-compose $args
