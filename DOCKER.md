# Docker Quick Start

## Development (Local Testing)

```bash
# Build images locally
docker-compose build

# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

## Production (Raspberry Pi)

See [DEPLOYMENT.md](DEPLOYMENT.md) for complete setup guide.

### Quick Deploy:
```bash
# 1. Copy .env.example to .env and edit
cp .env.example .env

# 2. Login to GitHub Container Registry
echo YOUR_GITHUB_TOKEN | docker login ghcr.io -u YOUR_USERNAME --password-stdin

# 3. Deploy Watchtower
docker run -d --name watchtower --restart unless-stopped \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_POLL_INTERVAL=300 \
  containrrr/watchtower

# 4. Start app
docker-compose up -d
```

Now every push to GitHub will auto-deploy to your Pi! 🚀

## FREE Services Used:
- ✅ GitHub Actions (2000 min/month free)
- ✅ GitHub Container Registry (unlimited free)
- ✅ Watchtower (open source)

Total cost: **$0/month**
