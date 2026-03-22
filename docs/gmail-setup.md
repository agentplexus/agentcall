# Gmail Setup Guide

This guide walks through setting up Gmail as a chat provider in AgentComms.

## Overview

AgentComms uses the Gmail API with OAuth 2.0 for authentication. This allows sending emails programmatically from your Gmail account without needing to enable "less secure apps" or use app-specific passwords.

## Prerequisites

- A Google account
- Access to Google Cloud Console

## Step 1: Create a Google Cloud Project

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click "Select a project" dropdown at the top
3. Click "New Project"
4. Enter a project name (e.g., "AgentComms")
5. Click "Create"

## Step 2: Enable the Gmail API

1. In the Cloud Console, go to **APIs & Services** > **Library**
2. Search for "Gmail API"
3. Click on "Gmail API"
4. Click **Enable**

## Step 3: Configure OAuth Consent Screen

1. Go to **APIs & Services** > **OAuth consent screen**
2. Select **External** (or Internal if using Google Workspace)
3. Click **Create**
4. Fill in the required fields:
   - **App name**: AgentComms
   - **User support email**: Your email
   - **Developer contact email**: Your email
5. Click **Save and Continue**
6. On the "Scopes" page, click **Add or Remove Scopes**
7. Find and select `https://www.googleapis.com/auth/gmail.send`
8. Click **Update**, then **Save and Continue**
9. Add yourself as a test user (required for external apps in testing)
10. Click **Save and Continue**, then **Back to Dashboard**

## Step 4: Create OAuth Credentials

1. Go to **APIs & Services** > **Credentials**
2. Click **Create Credentials** > **OAuth client ID**
3. Select **Desktop app** as the application type
4. Enter a name (e.g., "AgentComms CLI")
5. Click **Create**
6. Click **Download JSON** to download your credentials
7. Save the file as `~/.agentcomms/gmail_credentials.json`

## Step 5: Configure AgentComms

### Option A: Environment Variables

```bash
# Enable Gmail
export AGENTCOMMS_GMAIL_ENABLED=true

# Path to credentials file (required)
export AGENTCOMMS_GMAIL_CREDENTIALS_FILE=~/.agentcomms/gmail_credentials.json

# Path to store OAuth token (optional, defaults to ~/.agentcomms/gmail_token.json)
export AGENTCOMMS_GMAIL_TOKEN_FILE=~/.agentcomms/gmail_token.json

# From address (optional, defaults to "me" for authenticated user)
export AGENTCOMMS_GMAIL_FROM_ADDRESS=me
```

### Option B: JSON Configuration

Add to `~/.agentcomms/config.json`:

```json
{
  "chat": {
    "gmail": {
      "enabled": true,
      "credentials_file": "${HOME}/.agentcomms/gmail_credentials.json",
      "token_file": "${HOME}/.agentcomms/gmail_token.json",
      "from_address": "me"
    }
  }
}
```

## Step 6: Authorize the Application

On first run, AgentComms will open a browser window for OAuth authorization:

1. Start AgentComms: `agentcomms daemon`
2. A browser window opens to Google's OAuth consent screen
3. Sign in with your Google account
4. Review the permissions and click **Allow**
5. The browser shows "Authorization successful" and you can close it
6. AgentComms stores the token in your token file

After authorization, AgentComms won't need browser access again unless the token expires or is revoked.

## Usage

### Sending Emails via MCP

```json
{
  "tool": "send_message",
  "arguments": {
    "provider": "gmail",
    "chat_id": "recipient@example.com",
    "content": "Hello from AgentComms!",
    "metadata": {
      "subject": "Test Email"
    }
  }
}
```

### Email Subject

The email subject can be set in several ways:

1. **Metadata field** (recommended):
   ```json
   { "metadata": { "subject": "Your Subject Here" } }
   ```

2. **First line of content**: If no subject is provided, the first line of the content is used

3. **Default**: Falls back to "Message from AgentComms"

### HTML Emails

Set the format to HTML for rich content:

```json
{
  "tool": "send_message",
  "arguments": {
    "provider": "gmail",
    "chat_id": "recipient@example.com",
    "content": "<h1>Hello</h1><p>This is <strong>HTML</strong> content.</p>",
    "format": "html",
    "metadata": {
      "subject": "HTML Test"
    }
  }
}
```

## Troubleshooting

### "Access blocked: AgentComms has not completed the Google verification process"

This is normal for apps in testing mode. Solutions:

1. Add your email as a test user in OAuth consent screen
2. Or publish the app (requires verification for broader access)

### "Token has been expired or revoked"

Delete the token file and re-authorize:

```bash
rm ~/.agentcomms/gmail_token.json
agentcomms daemon  # Will prompt for re-authorization
```

### "Missing required environment variables"

Ensure `AGENTCOMMS_GMAIL_CREDENTIALS_FILE` points to a valid credentials JSON file.

### "Failed to read Gmail credentials file"

Check that the file exists and is readable:

```bash
cat ~/.agentcomms/gmail_credentials.json
```

The file should contain JSON with `installed` or `web` key containing `client_id` and `client_secret`.

## Security Notes

1. **Never commit credentials**: Add `gmail_credentials.json` and `gmail_token.json` to `.gitignore`

2. **Token storage**: The OAuth token file contains refresh tokens. Protect it like a password.

3. **Scope limitation**: AgentComms only requests `gmail.send` scope, which cannot read your emails.

4. **Revoke access**: You can revoke AgentComms access at any time from [Google Account Security](https://myaccount.google.com/permissions).

## API Quotas

Gmail API has usage limits:

| Quota | Limit |
|-------|-------|
| Messages sent per day | 500 (personal), 2000 (Workspace) |
| Requests per minute | 250 |

For higher limits, consider using a Google Workspace account.
