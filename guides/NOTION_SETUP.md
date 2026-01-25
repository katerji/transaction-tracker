# Notion Setup Guide - Transaction Tracker

Complete guide to set up your Notion database for the Go transaction tracker application.

---

## Why Notion?

✅ **Free for personal use** - Unlimited pages and databases
✅ **Simple authentication** - Just API key (no OAuth flow!)
✅ **Beautiful interface** - Clean, modern UI
✅ **Unlimited transactions** - No record limits
✅ **Great mobile app** - Track on the go

**Setup Time: ~10 minutes**

---

## Part 1: Create Notion Account

1. Go to https://www.notion.so/
2. Click **"Try Notion free"** or **"Sign up"**
3. Create account with email or Google
4. Verify your email

**Free Forever:** Personal plan gives you unlimited pages and databases!

---

## Part 2: Create Transaction Database

### Step 1: Create New Page

1. Once logged in, click **"+ New page"** in the sidebar
2. Name it: `💰 Transaction Tracker`
3. In the page, type `/database` and press Enter
4. Select **"Table - Inline"**
5. Name the database: `Transactions`

**What you should see:**
- A blank table with a "Name" column
- "+ New" button to add rows
- Property options in the header

---

### Step 2: Configure Database Properties

You need to add specific columns (properties) for the Go application to work correctly.

**How to add a property:**
1. Click the **"+"** button at the far right of the header row
2. Enter the property name
3. Select the property type from the dropdown

**Add these properties (in order):**

| Property Name | Type | Notes |
|---------------|------|-------|
| Name | Title | Already exists - stores transaction description |
| Transaction Date | Date | When the transaction occurred |
| Amount (AED) | Number | Transaction amount in AED |
| Category | Select | Transaction category (see below) |
| Confidence | Number | AI confidence score (0-100) |
| Billing Cycle | Text | Billing cycle (e.g., "Jan 2026") |
| Timestamp | Date | When added to database (include time) |

---

### Step 3: Configure Category Property (Critical!)

The **Category** property is a **Select** type and needs these exact options:

1. Click on the **Category** column header
2. Click **"Edit property"**
3. Add these options (click "+ Add an option" for each):

**Required Category Options:**
- `Food & Dining`
- `Transport`
- `Shopping`
- `Bills & Utilities`
- `Entertainment`
- `Health & Fitness`
- `Travel`
- `Cash Withdrawal`
- `Income/Transfer`
- `Unknown`

**Important:** Category names must match exactly (including spaces and capitalization) for the Go app to work properly.

**Optional:** Add colors to each category for visual organization
**Optional:** Add emojis: `🍔 Food & Dining`, `🚗 Transport`, etc.

---

### Step 4: Customize View (Optional)

**Sort by date:**
1. Click **"..."** at top right of database
2. Select **"Sort"** → **"+ Add a sort"**
3. Choose **"Transaction Date"**
4. Select **"Descending"** (newest first)

**Group by category:**
1. Click **"..."** → **"Group"**
2. Select **"Category"**
3. Transactions now grouped by category!

**Filter current cycle:**
1. Click **"..."** → **"Filter"** → **"+ Add a filter"**
2. Select **"Billing Cycle"**
3. Choose **"Contains"** → Type current month (e.g., "Jan 2026")

---

## Part 3: Create Notion Integration

### Step 1: Access Integrations

1. Go to https://www.notion.so/my-integrations
2. Or: Click profile icon (bottom left) → **"Settings & members"** → **"Integrations"** → **"Develop or manage integrations"**

---

### Step 2: Create New Integration

1. Click **"+ New integration"** button
2. Fill out the form:

**Basic Information:**
- **Name**: `Transaction Tracker` (or any name you like)
- **Associated workspace**: Select your workspace
- **Logo**: Optional (skip)

**Capabilities:**
- ✅ **Read content**
- ✅ **Update content**
- ✅ **Insert content**

**Other settings:**
- Leave defaults

3. Click **"Submit"**

---

### Step 3: Copy Integration Secret (API Token)

After creating the integration:

1. You'll see **"Internal Integration Secret"**
2. Click **"Show"** and then **"Copy"**
3. **SAVE THIS TOKEN** - you'll need it for the Go app

**Format:** `secret_abc123def456xyz789...`

**⚠️ Important:** Treat this like a password - don't share it publicly!

---

### Step 4: Share Database with Integration

**This is critical!** You must explicitly share your database with the integration.

1. Go back to your **Transaction Tracker** page in Notion
2. Click the **"..."** (three dots) at the top right
3. Scroll down and select **"Add connections"** or **"Connections"**
4. Search for your integration (e.g., `Transaction Tracker`)
5. Click on it to add the connection
6. Confirm: You should see **"[Integration name] can access this page"**

**Without this step, the Go app cannot access your database!**

---

### Step 5: Get Database ID

You need the database ID for the Go application configuration.

**Method 1: From URL**
1. Open your Transaction Tracker page in Notion
2. Click on the database to view it full-page
3. Look at the URL in your browser:
   ```
   https://www.notion.so/WORKSPACE_NAME/DATABASE_ID?v=VIEW_ID
   ```
4. Copy the `DATABASE_ID` part (long string of letters and numbers)
5. Remove any hyphens

**Example:**
```
URL: https://www.notion.so/myworkspace/abc123def456?v=xyz789
Database ID: abc123def456
```

**Method 2: Share Link**
1. Click **"Share"** at the top right of your database
2. Click **"Copy link"**
3. Extract the ID from the link (same as Method 1)

**Save this Database ID** - you'll add it to your `.env` file.

---

## Part 4: Configure Go Application

Now that Notion is set up, configure your Go application:

### Step 1: Create `.env` File

In your `transaction-tracker` directory:

```bash
cd ~/Development/transaction-tracker
cp .env.example .env
```

### Step 2: Edit `.env` File

Open `.env` and add your Notion credentials:

```bash
# Notion API Configuration
NOTION_API_KEY=secret_your_integration_secret_here
NOTION_DATABASE_ID=your_database_id_here

# OpenAI API Configuration (get from OpenAI)
OPENAI_API_KEY=sk-proj-your_openai_key_here

# Server Configuration (optional)
PORT=8080
```

**Replace:**
- `secret_your_integration_secret_here` with the Integration Secret from Part 3, Step 3
- `your_database_id_here` with the Database ID from Part 3, Step 5
- `sk-proj-your_openai_key_here` with your OpenAI API key

---

## Part 5: Test the Connection

### Test Locally

1. **Run the Go app:**
   ```bash
   cd ~/Development/transaction-tracker
   go run *.go
   ```

2. **Test with a transaction:**
   ```bash
   curl -X POST http://localhost:8080/transaction \
     -H "Content-Type: application/json" \
     -d '{"text": "Spent AED 50 at Starbucks"}'
   ```

3. **Check Notion:**
   - Open your Transaction Tracker in Notion
   - You should see a new entry!

4. **Test stats:**
   ```bash
   curl http://localhost:8080/stats
   ```

---

## Troubleshooting

### Issue: "Notion API error: unauthorized"

**Cause:** Integration doesn't have access to the database

**Solution:**
1. Go to your Transaction Tracker page in Notion
2. Click **"..."** → **"Connections"**
3. Verify your integration is listed
4. If not, add it again

---

### Issue: "Database not found"

**Cause:** Wrong Database ID

**Solution:**
1. Get the database ID again from the URL
2. Make sure you're using the database ID, not the page ID
3. Remove any hyphens from the ID
4. Update `.env` file

---

### Issue: "Invalid property value for Category"

**Cause:** Category options in Notion don't match exactly

**Solution:**
1. Check Category property in Notion
2. Ensure exact names:
   - `Food & Dining` (not "Food and Dining")
   - `Bills & Utilities` (not "Bills and Utilities")
   - etc.
3. Names are case-sensitive!

---

### Issue: Properties not saving

**Cause:** Property names don't match exactly

**Solution:**
Verify property names in Notion match exactly:
- `Name` (Title type)
- `Transaction Date` (Date type)
- `Amount (AED)` (Number type) - **Include the parentheses!**
- `Category` (Select type)
- `Confidence` (Number type)
- `Billing Cycle` (Text type)
- `Timestamp` (Date type)

---

## Database Maintenance

### Viewing Your Data

**On Desktop:**
- Open Notion in browser or app
- Navigate to Transaction Tracker
- View as table, timeline, or other views

**On Mobile:**
- Open Notion app
- Find Transaction Tracker
- View and edit transactions

---

### Exporting Data

**To export your data:**
1. Click **"..."** at top right
2. Select **"Export"**
3. Choose format:
   - **CSV** - For Excel/spreadsheet import
   - **PDF** - For readable document
   - **HTML** - For web viewing
   - **Markdown** - For text format

---

### Adding Custom Views

**Create a "Current Month" view:**
1. Click **"+"** next to existing view
2. Name it "Current Month"
3. Add filter: `Billing Cycle` contains "Jan 2026"
4. Update filter each month

**Create a "By Category" view:**
1. Create new table view
2. Group by "Category"
3. Sort by "Amount (AED)" descending

**Create a "High Confidence" view:**
1. Create new table view
2. Add filter: `Confidence` > 90
3. See only highly confident categorizations

---

## Security & Best Practices

### Protecting Your Integration Secret

✅ **Do:**
- Store in `.env` file (not committed to git)
- Use environment variables
- Regenerate if compromised

❌ **Don't:**
- Commit to version control
- Share publicly
- Hardcode in source files

---

### Regenerating Integration Secret

If your secret is compromised:

1. Go to https://www.notion.so/my-integrations
2. Click your integration
3. Click **"Show"** next to Internal Integration Secret
4. Click **"Regenerate"**
5. Update `.env` file with new secret
6. Restart Go application

---

### Revoking Access

**To revoke integration access:**

1. Go to Transaction Tracker page in Notion
2. Click **"..."** → **"Connections"**
3. Find your integration
4. Click **"..."** next to it → **"Remove"**

Or delete the integration entirely:
1. Go to https://www.notion.so/my-integrations
2. Click your integration
3. Scroll to bottom → **"Delete integration"**

---

## Advanced: Multiple Databases

If you want separate databases (e.g., personal vs. business):

1. Create second database in Notion (follow Part 2)
2. Share with same integration (Part 3, Step 4)
3. Get second database ID
4. Run multiple instances of Go app with different `.env` files

---

## Summary

You now have:

✅ Notion account created
✅ Transaction database configured with all properties
✅ Notion integration created
✅ Database shared with integration
✅ API key and database ID obtained
✅ Go application configured with Notion credentials

**Next steps:**
1. Deploy your Go app to Fly.io (see README.md)
2. Set up phone shortcuts for easy transaction logging
3. Start tracking your expenses!

---

**Need help?**
- Notion API Docs: https://developers.notion.com
- Notion Help: https://www.notion.so/help
- Go App README: See `README.md` in transaction-tracker directory

---

**Version:** 2.0 (Go Application Edition)
**Last Updated:** January 2026
