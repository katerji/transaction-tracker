# HTTP Shortcuts Setup Guide - Transaction Tracker

Complete guide to set up transaction tracking from your phone using HTTP Shortcuts app with local n8n.

---

## Overview

**What you'll have:**
- One-tap transaction logging from your phone
- Works on your local WiFi network (100% local, no internet needed)
- Instant confirmation with transaction details
- Stats button to check spending
- Free forever, no subscriptions

**Setup Time:** 10 minutes

---

## Part 1: Set Up n8n Workflows

### Step 1: Import Workflows

1. Open your n8n instance (e.g., `http://localhost:5678`)
2. Go to **Workflows**
3. Click **"Import from File"**
4. Import **`webhook-transaction-logging-notion.json`**
5. Import **`webhook-stats-notion.json`**

You should now have 2 workflows.

---

### Step 2: Configure Credentials

**You need 2 credentials:**

#### A) OpenAI API
1. Go to https://platform.openai.com/api-keys
2. Create new API key
3. In n8n, click on **"OpenAI Parse Transactions"** node
4. Add credential with your API key
5. Save as `OpenAI account`

#### B) Notion API
1. Follow the Notion setup guide (`NOTION_SETUP_GUIDE.md`)
2. Get your Notion API token
3. In n8n, click on **"Add to Notion Database"** node
4. Add credential with your Notion token
5. Select your Transaction database
6. Save as `Notion account`

---

### Step 3: Get Webhook URLs

**For Transaction Logging Workflow:**

1. Open the **"Webhook Transaction Logging (Notion)"** workflow
2. Click on the **"Webhook Trigger"** node
3. Look for the **"Webhook URLs"** section
4. You'll see something like:
   ```
   Test URL: http://localhost:5678/webhook-test/transaction
   Production URL: http://localhost:5678/webhook/transaction
   ```

5. **Copy the Production URL**

**Important:** Replace `localhost` with your computer's local IP address.

---

### Step 4: Find Your Computer's Local IP Address

#### On Mac:
1. Open **System Settings** → **Network**
2. Click on your active connection (WiFi or Ethernet)
3. Look for **IP Address**: `192.168.x.x` or `10.0.x.x`

#### On Windows:
1. Open **Command Prompt**
2. Type: `ipconfig`
3. Look for **IPv4 Address** under your active network adapter

#### On Linux:
1. Open terminal
2. Type: `ip addr` or `ifconfig`
3. Look for IP address (usually starts with `192.168` or `10.0`)

**Example IP:** `192.168.1.100`

---

### Step 5: Build Your Webhook URLs

Replace `localhost` with your IP address:

**Transaction URL:**
```
http://192.168.1.100:5678/webhook/transaction
```

**Stats URL:**
```
http://192.168.1.100:5678/webhook/stats
```

**Save these URLs** - you'll need them for the phone app!

---

### Step 6: Activate Workflows

1. Open **"Webhook Transaction Logging (Notion)"**
2. Toggle **Active** to ON (top right)
3. Open **"Webhook Stats (Notion)"**
4. Toggle **Active** to ON

Both workflows should now be running!

---

## Part 2A: Android Setup (HTTP Shortcuts)

### Step 1: Install App

1. Open **Google Play Store**
2. Search for **"HTTP Shortcuts"** by Roland Kleger
3. Install the app (it's free)

---

### Step 2: Create "Log Transaction" Shortcut

1. Open **HTTP Shortcuts** app
2. Tap the **"+"** button (bottom right)
3. Tap **"Regular Shortcut"**

**Configure the shortcut:**

#### Basic Settings
- **Name**: `Log Transaction`
- **Icon**: Choose the 💰 or 📝 icon

#### Request Settings
1. Tap **"Request"**
2. **Method**: Select **POST**
3. **URL**: Paste your transaction webhook URL
   ```
   http://192.168.1.100:5678/webhook/transaction
   ```
   (Replace with your IP!)

4. **Request Body**:
   - Tap **"Request Body"**
   - Select **"Custom Body"**
   - **Content Type**: `application/json`
   - **Body**: Paste this:
   ```json
   {
     "text": "{input}"
   }
   ```

5. **Input Prompt**:
   - Go back to main shortcut settings
   - Tap **"Input/Dialogs"**
   - Enable **"Prompt for Input"**
   - **Input Title**: `Transaction Details`
   - **Input Message**: `Paste SMS transaction text`
   - **Input Type**: `Text (multiple lines)`

#### Response Handling (Optional but Nice)
1. Tap **"Response Handling"**
2. Tap **"Success Message"**
3. Select **"Show as Dialog"**
4. **Message**: `{response_data}`

This will show the confirmation from n8n!

5. Tap **"Done"** or **"✓"** to save

---

### Step 3: Create "Show Stats" Shortcut

1. Tap the **"+"** button again
2. Tap **"Regular Shortcut"**

**Configure:**

#### Basic Settings
- **Name**: `Show Stats`
- **Icon**: Choose 📊 icon

#### Request Settings
1. Tap **"Request"**
2. **Method**: Select **GET**
3. **URL**: Paste your stats webhook URL
   ```
   http://192.168.1.100:5678/webhook/stats
   ```

#### Response Handling
1. Tap **"Response Handling"**
2. Tap **"Success Message"**
3. Select **"Show as Dialog"**
4. **Message**: `{response_data}`

5. Tap **"Done"** to save

---

### Step 4: Add to Home Screen (Optional)

**For quick access:**

1. Long-press on a shortcut in HTTP Shortcuts
2. Tap **"Place on Home Screen"**
3. The shortcut appears as an icon on your home screen
4. One tap to open!

Repeat for both shortcuts.

---

### Step 5: Create Widget (Optional)

**For even faster access:**

1. Long-press on your Android home screen
2. Tap **"Widgets"**
3. Find **"HTTP Shortcuts"**
4. Drag the widget to your home screen
5. Select which shortcut to show
6. Resize if needed

Now you have one-tap transaction logging!

---

## Part 2B: iOS Setup (Shortcuts App)

### Step 1: Open Shortcuts App

The **Shortcuts** app comes pre-installed on iOS.

1. Open **Shortcuts** app
2. Tap **"+"** (top right) to create new shortcut

---

### Step 2: Create "Log Transaction" Shortcut

#### Add Actions:

1. **Ask for Input**:
   - Tap **"Add Action"**
   - Search for **"Ask for Input"**
   - Tap it to add
   - **Prompt**: `Paste transaction SMS text`
   - **Input Type**: `Text`
   - **Allow Multiple Lines**: ON

2. **Get Contents of URL**:
   - Tap **"+"** to add another action
   - Search for **"Get Contents of URL"**
   - Tap it to add
   - **URL**: Paste your transaction webhook URL
     ```
     http://192.168.1.100:5678/webhook/transaction
     ```
   - **Method**: `POST`
   - Tap **"Show More"**
   - **Request Body**: `JSON`
   - **Key**: `text`
   - **Value**: Tap and select **"Provided Input"** (from Ask for Input)

3. **Show Result**:
   - Tap **"+"** to add another action
   - Search for **"Show Result"**
   - Tap it to add
   - **Text**: Tap and select **"Contents of URL"**

4. **Name the Shortcut**:
   - Tap **"Untitled Shortcut"** at the top
   - Rename to: `Log Transaction`
   - Tap **"Done"**

---

### Step 3: Create "Show Stats" Shortcut

1. Tap **"+"** to create another shortcut

#### Add Actions:

1. **Get Contents of URL**:
   - Add **"Get Contents of URL"** action
   - **URL**: Paste your stats webhook URL
     ```
     http://192.168.1.100:5678/webhook/stats
     ```
   - **Method**: `GET`

2. **Show Result**:
   - Add **"Show Result"** action
   - **Text**: Select **"Contents of URL"**

3. **Name the Shortcut**:
   - Rename to: `Show Stats`
   - Tap **"Done"**

---

### Step 4: Add to Home Screen (iOS)

1. Long-press on the shortcut
2. Tap **"Details"**
3. Tap **"Add to Home Screen"**
4. Choose an icon and color
5. Tap **"Add"**

Now you have a home screen icon for quick access!

---

### Step 5: Add to Widgets (iOS)

1. Long-press on home screen
2. Tap **"+"** (top left)
3. Search for **"Shortcuts"**
4. Select a widget size
5. Tap **"Add Widget"**
6. Long-press the widget to configure
7. Select your transaction or stats shortcut

---

## Part 3: Using the System

### Logging a Transaction

**Android (HTTP Shortcuts):**
1. Receive SMS from bank: `"Your account was debited AED 50 at Starbucks"`
2. Copy the SMS text
3. Tap **"Log Transaction"** shortcut (home screen or widget)
4. Paste the text
5. Tap **"Send"** or **"OK"**
6. See confirmation dialog with details!

**iOS (Shortcuts):**
1. Copy SMS text
2. Tap **"Log Transaction"** shortcut
3. Paste when prompted
4. See the result!

---

### Checking Stats

**Either platform:**
1. Tap **"Show Stats"** shortcut
2. View your spending breakdown instantly!

---

### Example Workflow

**Morning:**
```
Received SMS: "AED 15 Careem ride"
→ Tap widget
→ Paste
→ Send
→ Confirmation: "✅ Added 1 transaction!
   Careem ride - 15.00 AED (Transport)"
```

**Lunch:**
```
SMS: "AED 45 at McDonald's"
→ Tap widget
→ Paste
→ Done
```

**Evening - Check spending:**
```
→ Tap "Show Stats"
→ See: "Total Spent: 60.00 AED
       Transport: 15 AED
       Food: 45 AED"
```

---

## Troubleshooting

### Issue: "Could not connect" or "Network Error"

**Causes:**
1. Phone not on same WiFi as computer
2. n8n not running
3. Wrong IP address
4. Firewall blocking connection

**Solutions:**

1. **Check WiFi**: Ensure phone and computer are on the same WiFi network

2. **Verify n8n is running**:
   - Open browser on computer
   - Go to `http://localhost:5678`
   - Should see n8n interface

3. **Test webhook from computer**:
   - Open browser
   - Go to: `http://192.168.1.100:5678/webhook/stats` (use your IP)
   - Should see JSON response

4. **Check firewall**:

   **Mac:**
   ```
   System Settings → Network → Firewall
   - Turn off firewall temporarily to test
   - Or add n8n to allowed apps
   ```

   **Windows:**
   ```
   Windows Defender Firewall → Allow an app
   - Add Node.js or n8n
   ```

   **Linux:**
   ```bash
   sudo ufw allow 5678
   ```

5. **Try computer's IP from phone browser**:
   - Open browser on phone
   - Go to: `http://192.168.1.100:5678/webhook/stats`
   - Should see JSON response
   - If yes: Connection works, check shortcut configuration
   - If no: Network/firewall issue

---

### Issue: Webhook URL Changes

**Cause:** Computer's IP address changed (DHCP)

**Solutions:**

**Option 1: Use hostname (Mac only)**
- Replace IP with computer name: `http://YourMacName.local:5678/webhook/transaction`
- Find computer name: System Settings → Sharing

**Option 2: Set Static IP**
- Configure router to assign fixed IP to your computer
- Or set static IP in network settings

**Option 3: Update shortcuts when IP changes**
- Find new IP address
- Edit shortcuts with new URL

---

### Issue: Response shows raw JSON

**Cause:** Response handling not configured properly

**Android Solution:**
1. Edit shortcut
2. Go to Response Handling
3. Enable **"Show as Dialog"**
4. Message: `{response_data}`

**iOS Solution:**
1. Edit shortcut
2. Add **"Show Result"** action
3. Select the API response

---

### Issue: Transaction not saved to Notion

**Causes:**
1. Notion credentials not configured
2. Database not selected
3. Database not shared with integration

**Solutions:**
1. Check n8n execution log for errors
2. Verify Notion credential in n8n
3. Ensure database is selected in "Add to Notion Database" node
4. Check database is shared with your integration (see Notion guide)

---

### Issue: Can't access from outside home WiFi

**This is expected!** The setup is designed for local-only access.

**If you need remote access:**
1. Use a VPN to connect to your home network
2. Or consider using a tunnel service (but this adds complexity and security considerations)

---

## Advanced Tips

### Batch Logging Multiple Transactions

Send multiple SMS texts at once:

```
Transaction 1: AED 50 Starbucks
Transaction 2: AED 15 Careem
Transaction 3: USD 10 Amazon
```

The AI will parse all 3 and save them!

---

### Custom Categories

Edit the OpenAI prompt in the n8n workflow to add your own categories.

---

### Notification on Save

Add a notification node in n8n to get a phone notification when transaction is saved.

---

### Auto-backup

Create a scheduled workflow in n8n to export Notion data daily.

---

## Security Considerations

### Local Network Only

✅ **Pros:**
- Data never leaves your network
- No external services (except OpenAI for parsing)
- Fast and reliable
- No monthly costs

⚠️ **Cons:**
- Only works on home WiFi
- Need VPN for remote access

### Securing Your Setup

1. **Strong WiFi password**: Prevent unauthorized network access
2. **n8n authentication**: Enable in n8n settings
3. **Keep n8n updated**: Regular security updates
4. **Backup Notion data**: Regular exports

### What Data Goes Where

**Stays Local (on your network):**
- Webhook requests/responses
- n8n processing

**Sent to External Services:**
- OpenAI: SMS text for parsing (via HTTPS)
- Notion: Parsed transaction data (via HTTPS API)

**Never Stored:**
- SMS texts are processed then discarded
- No logs kept on external servers

---

## Alternative: Using from Computer

You can also trigger these webhooks from your computer!

### Using curl (Terminal/Command Prompt):

**Log transaction:**
```bash
curl -X POST http://localhost:5678/webhook/transaction \
  -H "Content-Type: application/json" \
  -d '{"text": "AED 50 at Starbucks"}'
```

**Get stats:**
```bash
curl http://localhost:5678/webhook/stats
```

### Using Browser:

**Stats (GET request):**
Just open in browser:
```
http://localhost:5678/webhook/stats
```

**Transaction (POST request):**
Create a simple HTML form and open it in browser.

---

## What's Next?

**Enhancements you can add:**
- 📸 OCR: Scan receipt images and extract amounts
- 🔔 Notifications: Push notification on successful save
- 📅 Scheduled reports: Daily/weekly spending emails
- 💡 Budget alerts: Notify when category exceeds budget
- 🎯 Savings goals: Track progress toward goals
- 👥 Family sharing: Multiple users logging transactions

---

## Summary

You now have a fully functional, 100% local transaction tracking system:

✅ **One-tap logging from phone**
✅ **Works on local WiFi (no internet required)**
✅ **Instant confirmation**
✅ **Stats on demand**
✅ **Private & secure**
✅ **Free forever**

**Workflow:**
```
Phone → HTTP Shortcuts → Local n8n → OpenAI (parse) → Notion → Response
```

**Enjoy effortless expense tracking from your phone! 📱💰**

---

**Version:** 1.0 (HTTP Shortcuts Edition)
**Last Updated:** January 2026
**Platforms:** Android (HTTP Shortcuts) + iOS (Shortcuts app)
