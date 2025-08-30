# Setting Up NPM Token for Automated Publishing

This guide explains how to set up an NPM token so GitHub Actions can automatically publish to npm.

## Step 1: Generate NPM Token

### Option A: Via NPM CLI (Recommended)

1. **Login to npm**:
```bash
npm login
# Enter your username: kubeorchestra
# Enter your password
# Enter your email
```

2. **Create an automation token**:
```bash
npm token create --read-only=false
```

Output will look like:
```
┌────────────────┬──────────────────────────────────────┐
│ token          │ npm_xxxxxxxxxxxxxxxxxxxxxxxxxxxxx   │
│ cidr_whitelist │                                      │
│ readonly       │ false                                │
│ automation     │ true                                 │
│ created        │ 2024-01-01T00:00:00.000Z           │
└────────────────┴──────────────────────────────────────┘
```

3. **Copy the token** starting with `npm_`

### Option B: Via NPM Website

1. Go to: https://www.npmjs.com/
2. Login with your account
3. Click your profile icon → **Access Tokens**
4. Click **Generate New Token**
5. Select **Automation** (Important!)
6. Give it a name like "GitHub Actions - OrchCLI"
7. Click **Generate Token**
8. Copy the token immediately (you won't see it again!)

## Step 2: Add Token to GitHub Repository

1. **Go to your repository**: https://github.com/KubeOrchestra/cli

2. **Navigate to Settings**:
   - Click **Settings** tab
   - Scroll down to **Security** section
   - Click **Secrets and variables** → **Actions**

3. **Add the secret**:
   - Click **New repository secret**
   - Name: `NPM_TOKEN`
   - Value: Paste your npm token (npm_xxxx...)
   - Click **Add secret**

## Step 3: Verify Setup

The token is now available to GitHub Actions as `${{ secrets.NPM_TOKEN }}`

To test:
1. Push a new tag: `git tag v0.0.3 && git push origin v0.0.3`
2. Check Actions tab: https://github.com/KubeOrchestra/cli/actions
3. Watch the "Release and Publish" workflow

## Token Permissions

The automation token needs these permissions:
- **Read** access to packages
- **Publish** access to packages
- **No 2FA requirement** (automation tokens bypass 2FA)

## Security Notes

1. **Never commit tokens**: Always use GitHub secrets
2. **Use automation tokens**: Not personal tokens
3. **Rotate regularly**: Create new tokens periodically
4. **Limit scope**: Only give necessary permissions

## Troubleshooting

### Error: 401 Unauthorized
- Token is invalid or expired
- Create a new token and update secret

### Error: 403 Forbidden
- You don't have permission to publish to @kubeorchestra
- Make sure you're logged in as kubeorchestra or have access

### Error: 409 Conflict
- Version already exists on npm
- Bump version and try again

### Check Token Validity
```bash
# Set token as environment variable
export NPM_TOKEN=npm_xxxxx

# Check if token works
npm whoami --registry https://registry.npmjs.org/ --//registry.npmjs.org/:_authToken=$NPM_TOKEN
```

## Alternative: Manual Publishing

If you can't set up automated publishing:

1. **After GitHub release is created**:
```bash
# Pull latest with tag
git pull --tags

# Checkout the tag
git checkout v0.0.2

# Update package.json version
npm version 0.0.2 --no-git-tag-version

# Build package
make npm-build

# Login to npm
npm login

# Publish
npm publish --access public
```

## Using Different NPM Registry

If using a private registry:

1. Update `.github/workflows/release-and-publish.yml`:
```yaml
- name: Setup Node.js
  uses: actions/setup-node@v4
  with:
    node-version: '18'
    registry-url: 'https://your-registry.com'
```

2. Update package.json:
```json
"publishConfig": {
  "registry": "https://your-registry.com"
}
```

## Organization Setup

To publish under @kubeorchestra:

1. **Create organization** (if not exists):
   - Go to: https://www.npmjs.com/org/create
   - Name: kubeorchestra

2. **Add members** (optional):
   - Go to org settings
   - Invite collaborators

3. **Package naming**:
   - Must use: `@kubeorchestra/package-name`
   - Set in package.json: `"name": "@kubeorchestra/cli"`

## Summary

Once NPM_TOKEN is set up:
1. Push tag → GitHub Actions runs
2. Builds all binaries
3. Creates GitHub release
4. Publishes to npm automatically
5. Users can `npm install -g @kubeorchestra/cli`

No manual intervention needed!