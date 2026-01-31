# 🚀 Raspberry Pi Setup Guide

**Note: This is for later when you're ready to deploy to the Pi!**

## 📋 What You'll Need

- Raspberry Pi (3/4/5) with Raspberry Pi OS
- Internet connection
- SSH access to Pi
- Your GitHub username

## 🔧 Complete Setup Steps

### 1. Install Docker on Pi

```bash
# Install Docker
curl -fsSL https://get.docker.com | sh

# Add your user to docker group (no sudo needed)
sudo usermod -aG docker $USER

# Log out and back in for group to take effect
logout
# Then SSH back in
```

Verify Docker works:
```bash
docker --version
docker ps
```

### 2. Mount Your NAS (if external)

If your NAS is external, mount it first:

```bash
# Create mount point
sudo mkdir -p /mnt/music

# For CIFS/SMB (Windows share or NAS)
sudo mount -t cifs //NAS_IP/share /mnt/music -o username=USER,password=PASS

# For NFS
sudo mount -t nfs NAS_IP:/path/to/music /mnt/music

# Make it permanent (add to /etc/fstab)
sudo nano /etc/fstab
# Add line:
# //NAS_IP/share /mnt/music cifs username=USER,password=PASS 0 0
```

**OR** if NAS is directly connected (USB drive):
```bash
# Find your drive
lsblk

# Mount it (replace sdX1 with your drive)
sudo mkdir -p /mnt/music
sudo mount /dev/sdX1 /mnt/music

# Make permanent
sudo nano /etc/fstab
# Add: /dev/sdX1 /mnt/music ext4 defaults 0 2
```

### 3. Clone Repository

```bash
cd ~
git clone https://github.com/YOUR-USERNAME/doyou-world-scraper.git
cd doyou-world-scraper
```

### 4. Configure Environment

```bash
# Copy example env file
cp .env.example .env

# Edit with your settings
nano .env
```

Configure these in `.env`:
```env
GITHUB_REPOSITORY_OWNER=your-github-username

# Ports (change if occupied)
API_PORT=3000
FRONTEND_PORT=8080

# NAS path (where you mounted it)
NAS_PATH=/mnt/music

# Your timezone
TZ=Europe/Athens
```

### 5. Login to GitHub Container Registry

You need a Personal Access Token (PAT):

1. Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Give it a name: "Pi Docker Access"
4. Select scopes:
   - ✅ `read:packages` (required)
   - ✅ `repo` (if private repository)
5. Generate and **copy the token**

Then on Pi:
```bash
echo YOUR_GITHUB_TOKEN | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

### 6. Deploy Watchtower (Auto-Updater)

This will automatically pull new images when you push to GitHub:

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

**What it does:**
- Checks every 5 minutes (300 seconds)
- Pulls new images automatically
- Restarts containers with new versions
- Cleans up old images

### 7. Start Your Application

```bash
docker-compose up -d
```

### 8. Verify Everything Works

```bash
# Check containers are running
docker-compose ps

# Should show:
# doyou-api       running
# doyou-frontend  running

# View logs
docker-compose logs -f

# Check Watchtower
docker logs watchtower
```

### 9. Access Your App

Open in browser:
- **Frontend**: `http://YOUR_PI_IP:8080`
- **API**: `http://YOUR_PI_IP:3000/api/v1/health`

## 🎉 You're Done!

Now whenever you:
1. Make changes on your Mac
2. Push to GitHub
3. GitHub Actions builds new images (~5 minutes)
4. Watchtower detects and updates Pi (~5 minutes)

**Total auto-deploy time: ~10 minutes** ⚡

## 🔍 Troubleshooting

### Check if ports are occupied:
```bash
# Check what's using port 3000
sudo lsof -i :3000

# Check port 8080
sudo lsof -i :8080

# If occupied, change in .env file
```

### Container won't start:
```bash
# View detailed logs
docker-compose logs api
docker-compose logs frontend

# Restart everything
docker-compose down
docker-compose up -d
```

### Watchtower not updating:
```bash
# Check Watchtower logs
docker logs watchtower

# Look for "Found new" messages

# Force check now
docker restart watchtower

# Manually pull and restart
docker-compose pull
docker-compose up -d
```

### NAS not accessible:
```bash
# Check if mounted
df -h | grep music

# Check permissions
ls -la /mnt/music

# Remount if needed
sudo mount -a
```

### Out of disk space:
```bash
# Clean old Docker images
docker system prune -a

# Check disk usage
df -h
```

## 🛠️ Useful Commands

```bash
# Stop app
docker-compose down

# Restart app
docker-compose restart

# View logs (follow)
docker-compose logs -f

# View specific service logs
docker-compose logs -f api

# Update to latest manually
docker-compose pull
docker-compose up -d

# Check container stats
docker stats

# Clean up everything
docker-compose down -v
docker system prune -a
```

## 🔐 Security Tips

1. **Never commit `.env` file** - it's in .gitignore
2. **Keep GitHub token secure** - store safely
3. **Use strong passwords** for NAS mounts
4. **Consider using SSH keys** instead of passwords for NAS
5. **Update Pi regularly**: `sudo apt update && sudo apt upgrade`

## 🚀 Performance Tips for Pi

```bash
# If running Pi 3 or lower, reduce scan frequency:
# Edit watchtower interval to 600 (10 minutes)
docker run -d \
  --name watchtower \
  --restart unless-stopped \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -e WATCHTOWER_POLL_INTERVAL=600 \
  containrrr/watchtower
```

## 📱 Access from Outside Network (Optional)

If you want to access from internet:

1. **Use Tailscale** (easiest, free):
   ```bash
   curl -fsSL https://tailscale.com/install.sh | sh
   sudo tailscale up
   ```
   Then access via Tailscale IP

2. **Or use Cloudflare Tunnel** (free):
   - More complex but gives you HTTPS
   - See Cloudflare Tunnel docs

**DO NOT** expose ports directly without security!

## 💾 Backup Strategy

Backup these directories regularly:
```bash
# csv files
/home/pi/doyou-world-scraper/csv/

# downloads
/home/pi/doyou-world-scraper/downloads/

# Create backup script
nano backup.sh
```

```bash
#!/bin/bash
tar -czf backup-$(date +%Y%m%d).tar.gz csv/ downloads/
# Copy to NAS or external drive
cp backup-*.tar.gz /mnt/music/backups/
```

## 🔄 Update Strategy

**Automatic** (default):
- Watchtower handles everything
- Just push to GitHub

**Manual** (if you disable Watchtower):
```bash
cd ~/doyou-world-scraper
docker-compose pull
docker-compose up -d
```

## 📊 Monitoring

View real-time stats:
```bash
# All containers
docker stats

# Watchtower activity
docker logs -f watchtower | grep "Updating"

# App logs
docker-compose logs -f --tail=100
```

## 🆘 Emergency Recovery

If something breaks:
```bash
# Stop everything
docker-compose down

# Remove everything and start fresh
docker system prune -a -f
docker volume prune -f

# Re-pull images
docker-compose pull

# Start again
docker-compose up -d
```

---

**Remember:** This setup costs **$0/month** and auto-updates! 🎉
