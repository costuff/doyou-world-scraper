# Migrate DoYou + Owncast to Mac mini

This TODO contains the ordered steps and commands to move the services (DoYou scraper, Owncast, cloudflared tunnel, Watchtower, and stream-control) to the Mac mini where the external HDD now resides.

## Checklist

- [ ] Prepare Mac mini (install Docker, attach HDD)
- [ ] Copy repo and configs (`.env`, `docker-compose.prod.yml`, `cloudflared-config.yml`, `credentials.json`)
- [ ] Update paths and `.env` (set `NAS_PATH` to `/Volumes/<Name>`)
- [ ] Install `cloudflared` and configure tunnel on Mac mini
- [ ] Start `owncast`, `doyou-api`, `doyou-frontend`, and `stream-control`
- [ ] Install and run Watchtower
- [ ] Ensure auto-start on reboot (Docker restart policies + launchd/systemd equivalents)
- [ ] Test local services and public endpoints
- [ ] Cutover DNS / stop old tunnel and host

---

## Steps with commands

1) Prepare Mac mini

 - Install Docker (Docker Desktop) and Homebrew (if needed)

   ```bash
   # (macOS) install Homebrew if needed, then Docker Desktop and cloudflared
   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
   brew install --cask docker
   brew install cloudflare/cloudflare/cloudflared
   open --background -a Docker    # start Docker Desktop
   ```

 - Attach external HDD and note its mount path (example: `/Volumes/ExtHDD-8TB`).

2) Copy data & repo

 - From current machine, copy repository and cloudflared credentials securely to Mac mini:

   ```bash
   # on source machine
   cd ~/workspace
   tar -czf doyou-workspace.tgz doyou-world-scraper
   scp doyou-workspace.tgz user@macmini:/tmp/
   scp /etc/cloudflared/credentials.json user@macmini:/tmp/credentials.json
   # optionally rsync large HDD contents if you moved data files
   rsync -avh /mnt/mydrive/ExtHDD-8TB/ user@macmini:/Volumes/ExtHDD-8TB/
   ```

3) Prepare files on Mac mini

 - Extract and place files in project folder, update `.env` `NAS_PATH` to `/Volumes/<YourHDD>`.

   ```bash
   tar -xzf /tmp/doyou-workspace.tgz -C ~/workspace
   mkdir -p /etc/cloudflared
   sudo mv /tmp/credentials.json /etc/cloudflared/credentials.json
   sudo chown root:root /etc/cloudflared/credentials.json
   # edit .env
   sed -i '' 's|NAS_PATH=.*|NAS_PATH=/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks|' ~/workspace/doyou-world-scraper/.env
   ```

4) Configure and test `cloudflared` tunnel

 - Test in foreground first to ensure config works:

   ```bash
   cloudflared --config ~/workspace/doyou-world-scraper/cloudflared-config.yml tunnel run
   # or run in background as service
   cloudflared service install --config ~/workspace/doyou-world-scraper/cloudflared-config.yml
   sudo systemctl start cloudflared.service   # if service installed
   sudo systemctl enable cloudflared.service
   ```

5) Start Docker services

 - Pull images and bring up services with `docker-compose` from the project folder:

   ```bash
   cd ~/workspace/doyou-world-scraper
   export $(grep -v '^#' .env | xargs)
   docker-compose -f docker-compose.prod.yml pull
   docker-compose -f docker-compose.prod.yml up -d --force-recreate
   docker ps --filter "name=doyou" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
   ```

6) Install Watchtower

 - Start Watchtower to auto-update images:

   ```bash
   docker run -d \
     --name watchtower \
     -v /var/run/docker.sock:/var/run/docker.sock \
     containrrr/watchtower:latest \
     --cleanup --interval 300
   ```

7) Ensure auto-start on reboot

 - Docker containers: make sure restart policy is `unless-stopped` (compose already sets this). Enforce if needed:

   ```bash
   docker update --restart unless-stopped doyou-api doyou-frontend owncast doyou-cloudflared watchtower
   ```

 - `stream-control` (Flask app): either run under `launchd` (macOS) or package into a small Docker container. Quick `launchd` option: I can generate a plist if you want.

8) Test & cutover

 - Test local endpoints:

   - API: `http://localhost:${API_PORT:-3000}/api/v1/health`
   - Owncast web UI: `http://localhost:8080`
   - Stream control UI: `http://localhost:5555`

 - Test public hostnames (after `cloudflared` is running):

   - `https://radio.costuff.work`
   - `https://do-you-world-scraper.costuff.work`

 - When everything is confirmed, stop the old host's `cloudflared` tunnel and/or update DNS to point to the new instance.

---

If you want, I can now:

- generate a macOS `launchd` plist for `stream-control` and instructions to install it, or
- generate a script to copy files and start all services on the Mac mini, or
- walk you through testing `cloudflared` and performing cutover.

Tell me which of the three you want next and I'll produce the files/commands.
