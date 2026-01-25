# SQLite on Fly.io - Complete Guide

## How SQLite Works with Fly.io

### The Challenge
Fly.io machines are **ephemeral by default** - when they restart or redeploy, the file system is wiped. Since SQLite is a **file-based database**, you need persistent storage.

### The Solution: Fly.io Volumes
Fly.io provides **persistent volumes** that survive machine restarts and deployments.

---

## Current Configuration Analysis

Looking at your `fly.toml`:
```toml
[http_service]
  auto_stop_machines = 'stop'
  auto_start_machines = true
  min_machines_running = 0
```

**What this means:**
- ✅ **Single machine** - Perfect for SQLite (no multi-writer issues)
- ✅ **Auto-stop/start** - Machine sleeps when idle, wakes on request
- ✅ **Volume persists** - Even when machine stops, volume data remains

**Verdict:** Your current setup is **IDEAL for SQLite** ✅

---

## How to Configure SQLite with Fly.io Volumes

### Step 1: Create a Volume

```bash
# Create a 1GB volume (plenty for transaction data)
fly volumes create transactions_data --region sin --size 1

# Note: 'sin' is Singapore (your current region from fly.toml)
# Free tier includes 3GB total volume storage
```

### Step 2: Update fly.toml

Add volume mount configuration:

```toml
# Add this section to your fly.toml
[mounts]
  source = "transactions_data"
  destination = "/data"
```

**Full updated fly.toml:**
```toml
app = 'transaction-tracker-damp-bush-5483'
primary_region = 'sin'

[build]

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = 'stop'
  auto_start_machines = true
  min_machines_running = 0
  processes = ['app']

# Add this section
[mounts]
  source = "transactions_data"
  destination = "/data"

[[vm]]
  memory = '1gb'
  cpu_kind = 'shared'
  cpus = 1
  memory_mb = 1024
```

### Step 3: Update Your App to Use Volume Path

In your `.env`:
```bash
DATABASE_PATH=/data/transactions.db
```

Or in code (recommended):
```go
func getDatabasePath() string {
    // Use volume path if it exists (production on Fly.io)
    if _, err := os.Stat("/data"); err == nil {
        return "/data/transactions.db"
    }
    // Otherwise use local path (development)
    return "./transactions.db"
}
```

### Step 4: Deploy

```bash
fly deploy
```

**That's it!** Your SQLite database will now persist across deployments.

---

## How It Works

```
┌─────────────────────────────────────┐
│  Fly.io Machine (Ephemeral)        │
│                                     │
│  ┌──────────────────────────────┐  │
│  │  Your Go App                 │  │
│  │  - Reads/Writes to DB        │  │
│  └──────────┬───────────────────┘  │
│             │                       │
│             ▼                       │
│  ┌──────────────────────────────┐  │
│  │  /data/transactions.db       │◄─┼─── Mounted Volume
│  │  (SQLite file)               │  │     (Persistent!)
│  └──────────────────────────────┘  │
│                                     │
└─────────────────────────────────────┘

When machine restarts:
- App files → Wiped and rebuilt
- Volume (/data) → PERSISTS with all data intact
```

---

## Pros & Cons

### ✅ Pros

1. **Simple Setup**
   - No external database service needed
   - No connection strings or credentials
   - Just a file path

2. **Performance**
   - Extremely fast (local disk reads/writes)
   - No network latency
   - Sub-millisecond queries

3. **Cost**
   - Free on Fly.io (3GB volume included)
   - No separate database service fees

4. **Reliability**
   - Battle-tested (used in production by many apps)
   - ACID compliant
   - Automatic backups (via volume snapshots)

5. **Perfect for Your Use Case**
   - Single region deployment
   - Low-to-medium write volume
   - Simple queries
   - Single machine = no concurrency issues

### ❌ Cons (and Mitigations)

1. **Single Region Only**
   - **Issue:** Volume is tied to one region
   - **Your case:** You're only in Singapore (sin) ✅ Not a problem

2. **Cannot Scale Horizontally**
   - **Issue:** SQLite doesn't support multiple writers
   - **Your case:** Single machine setup ✅ Not a problem
   - **If needed:** Can use Fly's LiteFS (replicated SQLite)

3. **Manual Backups Required**
   - **Issue:** No automatic backups by default
   - **Mitigation:** Easy to add backup script
   ```bash
   # Backup command
   fly ssh console -C "sqlite3 /data/transactions.db .dump > /data/backup-$(date +%Y%m%d).sql"
   ```

4. **Volume Can Fill Up**
   - **Issue:** 1GB limit (default free tier)
   - **Your case:** ~1000 transactions = ~500KB ✅ 1GB lasts years
   - **Mitigation:** Can expand volume size anytime

---

## Volume Management

### Check Volume Status
```bash
fly volumes list
```

### Create Backup
```bash
# Method 1: SQLite dump via SSH
fly ssh console -C "sqlite3 /data/transactions.db .dump" > backup.sql

# Method 2: Volume snapshot
fly volumes snapshots list transactions_data
fly volumes snapshots create transactions_data
```

### Restore from Backup
```bash
# Upload backup file
fly ssh console
cat > /data/transactions.db < backup.sql
```

### Expand Volume (if needed)
```bash
fly volumes extend transactions_data -s 5  # Expand to 5GB
```

---

## Comparison: SQLite vs External DB on Fly.io

| Feature | SQLite + Volume | Supabase/External DB |
|---------|----------------|---------------------|
| **Setup Complexity** | Very Simple | Medium |
| **Cost (Free Tier)** | $0 | $0 |
| **Performance** | Excellent (local) | Good (network latency) |
| **Scalability** | Single region | Multi-region |
| **Maintenance** | Minimal | Provider manages |
| **Backup** | Manual | Automatic |
| **Multi-Instance** | No (need LiteFS) | Yes |
| **Your Use Case** | ✅ Perfect | ✅ Also works |

---

## Advanced: LiteFS for Multi-Region (Optional)

If you ever need to scale globally, Fly.io offers **LiteFS** - replicated SQLite:

```toml
# Future consideration - not needed now
[experimental]
  litefs = true
```

**LiteFS Features:**
- Primary/replica SQLite databases
- Automatic replication across regions
- Read from closest region
- Write to primary, auto-syncs to replicas

**When to use:** Only if you need multi-region deployment (not needed for your case)

---

## Recommendation for Your App

### ✅ Use SQLite with Fly.io Volumes

**Why:**
1. Your setup is single-machine (perfect for SQLite)
2. Simple transaction tracking (no complex queries)
3. Low write volume (~10-50 transactions/day)
4. Single region (Singapore)
5. No external dependencies = simpler architecture
6. Zero cost for database
7. Excellent performance

### Migration Steps (simplified):

1. **Create volume:**
   ```bash
   fly volumes create transactions_data --region sin --size 1
   ```

2. **Update fly.toml:**
   ```toml
   [mounts]
     source = "transactions_data"
     destination = "/data"
   ```

3. **Update code:** Use `/data/transactions.db` as database path

4. **Deploy:**
   ```bash
   fly deploy
   ```

**Total time:** ~30 minutes to setup, 2-3 hours to implement SQLite code

---

## Frequently Asked Questions

### Q: What happens when the machine stops/restarts?
**A:** Volume data persists. When machine wakes up, your database is intact.

### Q: Can I lose data?
**A:** Very unlikely. Volumes are replicated by Fly.io. But you should still do periodic backups.

### Q: What if I exceed 1GB?
**A:** Can expand volume anytime: `fly volumes extend transactions_data -s 5`

### Q: Can I switch back to Notion?
**A:** Yes! Keep the interface pattern - easy to swap storage backends.

### Q: How do I migrate existing Notion data?
**A:** Export from Notion CSV → Write import script → Load into SQLite

### Q: Performance vs Notion API?
**A:** SQLite is 10-100x faster (local disk vs API calls)

---

## Next Steps

Want me to:
1. **Implement SQLite migration** (create database.go, update main.go)
2. **Add volume configuration** to fly.toml
3. **Add automatic backup script**
4. **Create Notion data export/import tool**

Let me know which you'd like to tackle first!
