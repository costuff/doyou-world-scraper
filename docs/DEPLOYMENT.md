# 🚀 Deployment Guide - 100% FREE Setup

This guide shows you how to deploy DoYou World Scraper with automatic updates from GitHub - completely free!

## 📋 Prerequisites

- GitHub account (free)
- Raspberry Pi with Docker installed
- GitHub repository (can be private)

## 🔧 One-Time Setup

### 1. Configure GitHub Container Registry

Your repository is already configured to push images to GitHub Container Registry (ghcr.io) - it's completely free!

**No additional setup needed** - GitHub Actions will automatically:
- Build ARM64 images for your Raspberry Pi
- Build AMD64 images for development
- Push to ghcr.io (unlimited storage, free)

### 2. Setup on Raspberry Pi

#### Install Docker (if not already installed)
```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
# Log out and back in
```

#### Clone your repository
```bash
git clone https://github.com/YOUR-USERNAME/doyou-world-scraper.git
cd doyou-world-scraper
```

#### Create .env file
```bash
cp .env.example .env
nano .env
```

Edit these values:
```env
GITHUB_REPOSITORY_OWNER=your-github-username
API_PORT=3000         # Change if port 3000 is occupied
FRONTEND_PORT=8080    # Change if port 8080 is occupied
NAS_PATH=/mnt/music   # Your NAS mount point
TZ=Europe/Athens
```

#### Login to GitHub Container Registry
```bash
# Create a Personal Access Token (PAT) with read:packages permission
# Go to: GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
# Generate new token with 'read:packages' scope

echo YOUR_GITHUB_TOKEN | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

### 3. Deploy Watchtower (Auto-Update Container)

```bash
docker run -d \
  --name watchtower \
  --restart unless-stopped \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_POLL_INTERVAL=300 \
  -e WATCHTOWER_CLEANUP=true \
  -e WATCHTOWER_INCLUDE_RESTARTING=true \
  containrrr/watchtower
```

**Watchtower will now check for updates every 5 minutes** and automatically pull new images!

### 4. Start Your Application

```bash
docker-compose up -d
```

Check status:
```bash
docker-compose ps
docker-compose logs -f
```

## 🎉 That's It!

### How It Works:

1. **Push to GitHub** → GitHub Actions builds images (free 2000 min/month)
2. **Images pushed to ghcr.io** → Free unlimited storage
3. **Watchtower checks every 5 minutes** → Detects new images
4. **Auto-pulls and restarts** → Zero-downtime updates

### Access Your App:
:8080` (or your configured FRONTEND_PORT)
- API: `http://YOUR_PI_IP:3000` (or your configured API_PORT)

**Common Port Alternatives if Occupied:**
- Frontend: 8080, 8888, 3001, 5000
- API: 3001, 8000, 8001, 9000
- API: `http://YOUR_PI_IP:3000`

## 🔍 Monitoring

View logs:
```bash
docker-compose logs -f

# Just API
docker-compose logs -f api

# Just frontend
docker-compose logs -f frontend

# Watchtower
docker logs -f watchtower
```

Check Watchtower updates:
```bash
docker logs watchtower | grep "Found new"
```

## 🛠️ Useful Commands

```bash
# Stop everything
docker-compose down

# Restart
docker-compose restart

# Force pull latest images
docker-compose pull
docker-compose up -d

# View running containers
docker ps

# Clean up old images
docker image prune -a
```

## 🚨 Troubleshooting

### Images not updating?
```bash
# Check Watchtower logs
docker logs watchtower

# Manually trigger update
docker-compose pull
docker-compose up -d
```

### Permission issues?
```bash
# Make sure user is in docker group
sudo usermod -aG docker $USER
# Log out and back in
```

### Build failed on GitHub?
- Check Actions tab in your GitHub repo
- Free tier: 2000 minutes/month (plenty for this project)
- Each build uses ~3-5 minutes

## 💰 Cost Breakdown

- **GitHub Actions**: FREE (2000 min/month = ~400 builds)
- **GitHub Container Registry**: FREE (unlimited storage)
- **Watchtower**: FREE (open source)
- **Docker**: FREE

**Total: $0/month** ✨

## 🔐 Security Notes

- Keep your GitHub token secure
- Use `.env` file (never commit it)
- NAS volume is mounted read-only by default
- Containers run as non-root users

## 📚 Additional Configuration

### Custom Watchtower interval (seconds)
```bash
# Check every 2 minutes instead of 5
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_POLL_INTERVAL=120 \
  containrrr/watchtower
```

### Enable Watchtower notifications
```bash
# Add email notifications (optional)
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_NOTIFICATIONS=email \
  -e WATCHTOWER_NOTIFICATION_EMAIL_FROM=from@example.com \
  -e WATCHTOWER_NOTIFICATION_EMAIL_TO=to@example.com \
  containrrr/watchtower
```

## 🎯 CasaOS Integration

If using CasaOS:
1. Add via Custom Install
2. Use the docker-compose.yml
3. Configure environment variables in CasaOS UI
4. Watchtower will still work automatically!
