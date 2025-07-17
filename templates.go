package main

const indexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>DDNS Pilot Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #007bff; padding-bottom: 10px; margin-bottom: 20px; }
        .header h1 { color: #333; margin: 0; }
        .nav-buttons { display: flex; gap: 10px; }
        .btn { padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; display: inline-block; font-size: 14px; }
        .btn-primary { background-color: #007bff; color: white; }
        .btn-secondary { background-color: #6c757d; color: white; }
        .btn-success { background-color: #28a745; color: white; }
        .btn-danger { background-color: #dc3545; color: white; }
        .btn-warning { background-color: #ffc107; color: black; }
        .btn:hover { opacity: 0.8; }
        .status-panel { background: #e7f3ff; border: 1px solid #007bff; border-radius: 4px; padding: 15px; margin-bottom: 20px; }
        .status-panel h3 { margin-top: 0; color: #004085; }
        .status-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 10px; }
        .status-item { background: white; padding: 10px; border-radius: 4px; border: 1px solid #ddd; }
        .table-container { overflow-x: auto; }
        table { width: 100%; border-collapse: collapse; margin-top: 10px; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f8f9fa; font-weight: bold; }
        .record-name { font-weight: bold; color: #007bff; }
        .status-enabled { color: #28a745; font-weight: bold; }
        .status-disabled { color: #dc3545; font-weight: bold; }
        .last-ip { font-family: monospace; background: #f8f9fa; padding: 2px 4px; border-radius: 2px; }
        .actions { display: flex; gap: 5px; flex-wrap: wrap; }
        .empty-state { text-align: center; padding: 40px; color: #666; }
        .empty-state h3 { color: #999; }
        .alert { padding: 15px; margin-bottom: 20px; border: 1px solid transparent; border-radius: 4px; }
        .alert-success { background-color: #d4edda; border-color: #c3e6cb; color: #155724; }
        .alert-warning { background-color: #fff3cd; border-color: #ffeaa7; color: #856404; }
        .alert-error { background-color: #f8d7da; border-color: #f5c6cb; color: #721c24; }
        .alert-dismissible { position: relative; padding-right: 50px; }
        .alert .close { position: absolute; top: 15px; right: 15px; background: none; border: none; font-size: 18px; cursor: pointer; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🚁 DDNS Pilot</h1>
            <div class="nav-buttons">
                <a href="/add-record" class="btn btn-primary">Add Record</a>
                <a href="/settings" class="btn btn-secondary">Settings</a>
                <a href="/logout" class="btn btn-warning">Logout</a>
            </div>
        </div>

        {{if .UpdateMessage}}
        <div class="alert alert-{{.UpdateType}} alert-dismissible">
            {{.UpdateMessage | html}}
            <button type="button" class="close" onclick="this.parentElement.style.display='none'">&times;</button>
        </div>
        {{end}}

        <div class="status-panel">
            <h3>📊 Status Overview</h3>
            <div class="status-grid">
                <div class="status-item">
                    <strong>Current Public IP:</strong><br>
                    <span class="last-ip">{{.CurrentIP}}</span>
                </div>
                <div class="status-item">
                    <strong>Total Records:</strong><br>
                    {{len .Records}}
                </div>
                <div class="status-item">
                    <strong>Auto-Update:</strong><br>
                    {{if .Config.AutoUpdate}}✅ Enabled ({{.Config.UpdateInterval}}min){{else}}❌ Disabled{{end}}
                </div>
                <div class="status-item">
                    <form method="post" action="/update-records" style="margin: 0;">
                        <button type="submit" class="btn btn-success">🔄 Update All</button>
                    </form>
                </div>
            </div>
        </div>

        {{if .Records}}
        <div class="table-container">
            <table>
                <thead>
                    <tr>
                        <th>Record Name</th>
                        <th>Status</th>
                        <th>Proxied</th>
                        <th>Last IP</th>
                        <th>Last Updated</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Records}}
                    <tr>
                        <td class="record-name">{{.RecordName | html}}</td>
                        <td>
                            {{if .Enabled}}
                                <span class="status-enabled">✅ Enabled</span>
                            {{else}}
                                <span class="status-disabled">❌ Disabled</span>
                            {{end}}
                        </td>
                        <td>{{if .Proxied}}🟠 Yes{{else}}🔵 No{{end}}</td>
                        <td>
                            {{if .LastIP}}
                                <span class="last-ip">{{.LastIP | html}}</span>
                            {{else}}
                                <em>Never updated</em>
                            {{end}}
                        </td>
                        <td>
                            {{if .LastUpdated}}
                                {{.LastUpdated | html}}
                            {{else}}
                                <em>Never</em>
                            {{end}}
                        </td>
                        <td class="actions">
                            <form method="post" action="/update-single" style="display: inline;">
                                <input type="hidden" name="record_name" value="{{.RecordName | html}}">
                                <button type="submit" class="btn btn-success">Update</button>
                            </form>
                            <form method="post" action="/toggle-record" style="display: inline;">
                                <input type="hidden" name="record_name" value="{{.RecordName | html}}">
                                <button type="submit" class="btn {{if .Enabled}}btn-warning{{else}}btn-success{{end}}">
                                    {{if .Enabled}}Disable{{else}}Enable{{end}}
                                </button>
                            </form>
                            <a href="/edit-record?name={{.RecordName | urlquery}}" class="btn btn-secondary">Edit</a>
                            <form method="post" action="/remove-record" style="display: inline;" onsubmit="return confirm('Are you sure you want to remove this record?')">
                                <input type="hidden" name="record_name" value="{{.RecordName | html}}">
                                <button type="submit" class="btn btn-danger">Remove</button>
                            </form>
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        {{else}}
        <div class="empty-state">
            <h3>No DNS Records Configured</h3>
            <p>Get started by adding your first CloudFlare DNS record.</p>
            <a href="/add-record" class="btn btn-primary">Add Your First Record</a>
        </div>
        {{end}}
    </div>

    <script>
        // Auto-refresh page every 5 minutes if auto-update is enabled
        {{if .Config.AutoUpdate}}
        setTimeout(function() {
            window.location.reload();
        }, 5 * 60 * 1000);
        {{end}}
    </script>
</body>
</html>`

const loginTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Login - DDNS Pilot</title>
    <style>
        body { 
            font-family: Arial, sans-serif; 
            margin: 0; 
            padding: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .login-container { 
            background: white; 
            padding: 40px; 
            border-radius: 12px; 
            box-shadow: 0 8px 24px rgba(0,0,0,0.15);
            width: 100%;
            max-width: 400px;
            margin: 20px;
        }
        .logo-section {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo-section h1 { 
            color: #333; 
            margin: 0 0 10px 0;
            font-size: 28px;
            font-weight: bold;
        }
        .logo-section p {
            color: #666;
            margin: 0;
            font-size: 14px;
        }
        .security-warning {
            background-color: #fff3cd;
            border: 2px solid #ffeaa7;
            color: #856404;
            padding: 12px;
            border-radius: 6px;
            margin-bottom: 20px;
            font-size: 13px;
            text-align: center;
        }
        .security-warning strong {
            display: block;
            margin-bottom: 5px;
        }
        .form-group { 
            margin: 20px 0; 
        }
        label { 
            display: block; 
            margin-bottom: 8px; 
            font-weight: bold;
            color: #333;
        }
        input[type="text"], input[type="password"] { 
            width: 100%; 
            padding: 12px 16px; 
            border: 2px solid #e0e0e0; 
            border-radius: 6px; 
            box-sizing: border-box;
            font-size: 16px;
            transition: border-color 0.3s ease;
        }
        input[type="text"]:focus, input[type="password"]:focus {
            outline: none;
            border-color: #667eea;
        }
        .btn { 
            width: 100%;
            padding: 12px 20px; 
            border: none; 
            border-radius: 6px; 
            cursor: pointer; 
            font-size: 16px;
            font-weight: bold;
            transition: background-color 0.3s ease;
        }
        .btn-primary { 
            background-color: #667eea; 
            color: white; 
        }
        .btn-primary:hover { 
            background-color: #5a67d8; 
        }
        .btn-primary:disabled {
            background-color: #6c757d;
            cursor: not-allowed;
        }
        .error-message { 
            background-color: #f8d7da; 
            border: 1px solid #f5c6cb; 
            color: #721c24; 
            padding: 12px 16px; 
            border-radius: 6px; 
            margin-bottom: 20px;
            font-size: 14px;
        }
        .blocked-message {
            background-color: #f8d7da; 
            border: 2px solid #dc3545; 
            color: #721c24; 
            padding: 16px; 
            border-radius: 6px; 
            margin-bottom: 20px;
            font-size: 14px;
            text-align: center;
            font-weight: bold;
        }
        .footer {
            margin-top: 30px;
            text-align: center;
            font-size: 12px;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="login-container">
        <div class="logo-section">
            <h1>🚁 DDNS Pilot</h1>
            <p>CloudFlare Dynamic DNS Manager</p>
        </div>
        
        {{if .UsingDefaultPassword}}
        <div class="security-warning">
            <strong>⚠️ Security Notice</strong>
            <strong>Default credentials detected!</strong> Change the password after login.
            <br><br>
            • This tool manages your DNS records and CloudFlare API access<br>
            • Use only on trusted local networks<br>
            • Consider using on-demand for maximum security
        </div>
        {{else}}
        <div class="security-warning">
            <strong>⚠️ Security Notice</strong>
            This interface manages your DNS records and CloudFlare API access.
            <br><br>
            • Use only on trusted local networks<br>
            • Consider using on-demand for maximum security
        </div>
        {{end}}
        
        {{if .IsBlocked}}
        <div class="blocked-message">
            🚫 Too many failed attempts. Try again in 15 minutes.
        </div>
        {{else}}
            {{if .Error}}
            <div class="error-message">
                {{.Error}}
            </div>
            {{end}}
            
            <form method="post" action="/login" id="loginForm">
                <div class="form-group">
                    <label for="username">Username:</label>
                    <input type="text" id="username" name="username" value="admin" readonly>
                </div>
                
                <div class="form-group">
                    <label for="password">Password:</label>
                    <input type="password" id="password" name="password" required autofocus>
                </div>
                
                <button type="submit" class="btn btn-primary" id="loginBtn">
                    Sign In
                </button>
            </form>
        {{end}}
        
        <div class="footer">
            <p>DDNS Pilot v1.0</p>
        </div>
    </div>

    <script>
        {{if not .IsBlocked}}
        document.getElementById('loginForm').addEventListener('submit', function(e) {
            const btn = document.getElementById('loginBtn');
            btn.textContent = 'Signing in...';
            btn.disabled = true;
        });
        
        // Auto-focus on password field since username is readonly
        document.getElementById('password').focus();
        {{end}}
    </script>
</body>
</html>`

const changePasswordTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{if .Forced}}Required Password Change{{else}}Change Password{{end}} - DDNS Pilot</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); min-height: 100vh; display: flex; align-items: center; justify-content: center; }
        .container { max-width: 600px; width: 100%; background: white; border-radius: 8px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); padding: 2rem; }
        h1 { text-align: center; color: #333; margin-bottom: 2rem; }
        .form-group { margin-bottom: 1rem; }
        label { display: block; margin-bottom: 0.5rem; color: #555; font-weight: bold; }
        input[type="password"] { width: 100%; padding: 0.75rem; border: 1px solid #ddd; border-radius: 4px; font-size: 1rem; box-sizing: border-box; }
        input[type="password"]:focus { outline: none; border-color: #667eea; box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.25); }
        .btn { width: 100%; padding: 0.75rem; background: #667eea; color: white; border: none; border-radius: 4px; font-size: 1rem; cursor: pointer; margin-top: 1rem; }
        .btn:hover { background: #5a67d8; }
        .security-warning { background: #f8d7da; border: 1px solid #f5c6cb; border-radius: 4px; padding: 1.5rem; margin-bottom: 2rem; color: #721c24; }
        .security-warning h3 { margin-top: 0; color: #721c24; }
        .security-warning ul { margin: 1rem 0; padding-left: 1.5rem; }
        .security-warning li { margin-bottom: 0.5rem; font-weight: 500; }
        .acknowledgment { background: #fff3cd; border: 1px solid #ffc107; border-radius: 4px; padding: 1rem; margin: 1rem 0; color: #856404; }
        .acknowledgment label { display: flex; align-items: flex-start; cursor: pointer; font-weight: normal; }
        .acknowledgment input[type="checkbox"] { margin-right: 0.5rem; margin-top: 0.25rem; }
        .password-requirements { background: #e7f3ff; border: 1px solid #667eea; border-radius: 4px; padding: 1rem; margin-bottom: 1rem; color: #004085; }
        .password-requirements h4 { margin-top: 0; }
        .password-requirements ul { margin: 0.5rem 0; padding-left: 1.5rem; }
        {{if .Forced}}.forced-notice { background: #dc3545; color: white; padding: 1rem; margin-bottom: 2rem; border-radius: 4px; text-align: center; font-weight: bold; }{{end}}
    </style>
</head>
<body>
    <div class="container">
        <h1>{{if .Forced}}🚨 Required Password Change{{else}}🔐 Change Password{{end}}</h1>
        
        {{if .Forced}}
        <div class="forced-notice">
            You must change the default password before accessing DDNS Pilot!
        </div>
        {{end}}
        
        <div class="security-warning">
            <h3>🛡️ Security Information</h3>
            <p><strong>Please read and understand these security considerations:</strong></p>
            <ul>
                <li><strong>CloudFlare API Access:</strong> This tool stores and uses your CloudFlare API tokens</li>
                <li><strong>DNS Management:</strong> If compromised, an attacker could modify your DNS records</li>
                <li><strong>Local Use Only:</strong> Run only on trusted local networks (localhost/LAN)</li>
                <li><strong>On-Demand Usage:</strong> For maximum security, run only when needed</li>
                <li><strong>Secure Your API Tokens:</strong> Always maintain secure backups of your CloudFlare credentials</li>
            </ul>
        </div>

        <div class="password-requirements">
            <h4>Password Requirements:</h4>
            <ul>
                <li>Minimum 8 characters long</li>
                <li>Cannot be "admin" or other common passwords</li>
                <li>Choose a strong, unique password</li>
                <li>Consider using a password manager</li>
            </ul>
        </div>

        <form action="/change-password{{if .Forced}}?force=true{{end}}" method="post">
            <div class="form-group">
                <label for="new_password">New Password:</label>
                <input type="password" id="new_password" name="new_password" required minlength="8">
            </div>
            <div class="form-group">
                <label for="confirm_password">Confirm New Password:</label>
                <input type="password" id="confirm_password" name="confirm_password" required minlength="8">
            </div>
            
            <div class="acknowledgment">
                <label>
                    <input type="checkbox" name="security_acknowledged" required>
                    <span>I understand and acknowledge the security considerations outlined above. I will not expose this service to the internet and will use it responsibly on trusted networks only.</span>
                </label>
            </div>
            
            <button type="submit" class="btn">{{if .Forced}}Set New Password & Continue{{else}}Change Password{{end}}</button>
        </form>
    </div>

    <script>
        document.querySelector('form').addEventListener('submit', function(e) {
            var newPassword = document.getElementById('new_password').value;
            var confirmPassword = document.getElementById('confirm_password').value;
            var acknowledged = document.querySelector('input[name="security_acknowledged"]').checked;
            
            if (newPassword !== confirmPassword) {
                e.preventDefault();
                alert('Passwords do not match!');
                return;
            }
            
            if (newPassword.length < 8) {
                e.preventDefault();
                alert('Password must be at least 8 characters long!');
                return;
            }
            
            if (newPassword.toLowerCase() === 'admin') {
                e.preventDefault();
                alert('Cannot use "admin" as password!');
                return;
            }
            
            if (!acknowledged) {
                e.preventDefault();
                alert('You must acknowledge the security warnings to continue!');
                return;
            }
        });
    </script>
</body>
</html>`

const addRecordTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Add DNS Record - DDNS Pilot</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 2px solid #667eea; padding-bottom: 10px; }
        .form-group { margin: 15px 0; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input[type="text"], input[type="password"], textarea { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; }
        .btn { padding: 10px 20px; margin: 5px; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; display: inline-block; }
        .btn-primary { background-color: #667eea; color: white; }
        .btn-secondary { background-color: #6c757d; color: white; }
        .btn:hover { opacity: 0.8; }
        .checkbox-group { margin: 15px 0; }
        .checkbox-group input[type="checkbox"] { margin-right: 10px; }
        .help-text { font-size: 12px; color: #666; margin-top: 5px; }
        .info-box { background-color: #e7f3ff; border: 1px solid #667eea; padding: 15px; border-radius: 4px; margin-bottom: 20px; }
        .info-box h3 { margin-top: 0; color: #004085; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Add DNS Record</h1>
        
        <div class="info-box">
            <h3>📝 Instructions</h3>
            <p>Enter your CloudFlare DNS record details below. The zone and record IDs will be automatically looked up using your API token.</p>
            <ul>
                <li><strong>Record Name:</strong> Full domain name (e.g., home.example.com)</li>
                <li><strong>API Token:</strong> CloudFlare API token with DNS:Edit permissions</li>
                <li><strong>Proxied:</strong> Whether to proxy traffic through CloudFlare (orange cloud)</li>
            </ul>
        </div>
        
        <form method="post">
            <div class="form-group">
                <label>Record Name:</label>
                <input type="text" name="record_name" required placeholder="e.g., home.example.com">
                <div class="help-text">Full domain name for the DNS record</div>
            </div>
            
            <div class="form-group">
                <label>CloudFlare API Token:</label>
                <input type="password" name="api_token" required placeholder="Enter your CloudFlare API token" value="{{.DefaultAPIToken | html}}">
                <div class="help-text">API token with DNS:Edit permissions for your zone</div>
            </div>
            
            <div class="checkbox-group">
                <label>
                    <input type="checkbox" name="proxied" value="true">
                    Proxied via CloudFlare (Orange Cloud)
                </label>
                <div class="help-text">Enable CloudFlare proxy for this record</div>
            </div>
            
            <div class="form-group">
                <label>Notes (Optional):</label>
                <textarea name="notes" rows="3" placeholder="e.g., Home server, Office connection, etc."></textarea>
                <div class="help-text">Optional description for this record</div>
            </div>
            
            <button type="submit" class="btn btn-primary">Add Record</button>
            <a href="/" class="btn btn-secondary">Cancel</a>
        </form>
    </div>
</body>
</html>`

const editRecordTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Edit DNS Record - DDNS Pilot</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 2px solid #667eea; padding-bottom: 10px; }
        .form-group { margin: 15px 0; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input[type="text"], textarea { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; }
        input[readonly] { background-color: #f8f9fa; }
        .btn { padding: 10px 20px; margin: 5px; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; display: inline-block; }
        .btn-primary { background-color: #667eea; color: white; }
        .btn-secondary { background-color: #6c757d; color: white; }
        .btn:hover { opacity: 0.8; }
        .checkbox-group { margin: 15px 0; }
        .checkbox-group input[type="checkbox"] { margin-right: 10px; }
        .help-text { font-size: 12px; color: #666; margin-top: 5px; }
        .info-section { background-color: #f8f9fa; padding: 15px; border-radius: 4px; margin-bottom: 20px; }
        .info-section h3 { margin-top: 0; color: #495057; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Edit DNS Record</h1>
        
        <div class="info-section">
            <h3>📋 Record Information</h3>
            <p><strong>Record Name:</strong> {{.Record.RecordName | html}}</p>
            <p><strong>Created:</strong> {{.Record.CreatedAt | html}}</p>
            {{if .Record.LastUpdated}}<p><strong>Last Updated:</strong> {{.Record.LastUpdated | html}}</p>{{end}}
            {{if .Record.LastIP}}<p><strong>Current IP:</strong> {{.Record.LastIP | html}}</p>{{end}}
        </div>
        
        <form method="post">
            <div class="form-group">
                <label>Record Name:</label>
                <input type="text" value="{{.Record.RecordName | html}}" readonly>
                <div class="help-text">Record name cannot be changed</div>
            </div>
            
            <div class="checkbox-group">
                <label>
                    <input type="checkbox" name="proxied" value="true" {{if .Record.Proxied}}checked{{end}}>
                    Proxied via CloudFlare (Orange Cloud)
                </label>
                <div class="help-text">Enable/disable CloudFlare proxy for this record</div>
            </div>
            
            <div class="form-group">
                <label>Notes:</label>
                <textarea name="notes" rows="3" placeholder="e.g., Home server, Office connection, etc.">{{.Record.Notes | html}}</textarea>
                <div class="help-text">Optional description for this record</div>
            </div>
            
            <button type="submit" class="btn btn-primary">Save Changes</button>
            <a href="/" class="btn btn-secondary">Cancel</a>
        </form>
    </div>
</body>
</html>`

const settingsTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Settings - DDNS Pilot</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #667eea; padding-bottom: 10px; margin-bottom: 20px; }
        .header h1 { color: #333; margin: 0; }
        .form-group { margin: 15px 0; }
        label { display: block; margin-bottom: 5px; font-weight: bold; }
        input[type="number"], input[type="text"] { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; max-width: 200px; }
        .btn { padding: 10px 20px; margin: 5px; border: none; border-radius: 4px; cursor: pointer; text-decoration: none; display: inline-block; }
        .btn-primary { background-color: #667eea; color: white; }
        .btn-secondary { background-color: #6c757d; color: white; }
        .btn:hover { opacity: 0.8; }
        .checkbox-group { margin: 15px 0; }
        .checkbox-group input[type="checkbox"] { margin-right: 10px; }
        .help-text { font-size: 12px; color: #666; margin-top: 5px; }
        .settings-section { background-color: #f8f9fa; padding: 15px; border-radius: 4px; margin-bottom: 20px; border-left: 4px solid #667eea; }
        .settings-section h3 { margin-top: 0; color: #495057; }
        .warning-box { background-color: #fff3cd; border: 1px solid #ffc107; padding: 15px; border-radius: 4px; margin-bottom: 20px; color: #856404; }
        .warning-box h3 { margin-top: 0; color: #856404; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚙️ Settings</h1>
            <a href="/" class="btn btn-secondary">Back to Dashboard</a>
        </div>

        <div class="warning-box">
            <h3>⚠️ Important Notice</h3>
            <p>Changing the web port requires restarting the application. Changes to auto-update settings take effect immediately.</p>
        </div>
        
        <form method="post">
            <div class="settings-section">
                <h3>🔄 Auto-Update Settings</h3>
                
                <div class="checkbox-group">
                    <label>
                        <input type="checkbox" name="auto_update" value="true" {{if .Config.AutoUpdate}}checked{{end}}>
                        Enable Automatic Updates
                    </label>
                    <div class="help-text">Automatically update DNS records at specified intervals</div>
                </div>
                
                <div class="form-group">
                    <label>Update Interval (minutes):</label>
                    <input type="number" name="update_interval" value="{{.Config.UpdateInterval}}" min="1" max="1440">
                    <div class="help-text">How often to check and update DNS records (1-1440 minutes)</div>
                </div>
            </div>
            
            <div class="settings-section">
                <h3>☁️ CloudFlare Settings</h3>
                
                <div class="form-group">
                    <label>Default CloudFlare API Token:</label>
                    <input type="text" name="default_api_token" value="{{.Config.DefaultAPIToken | html}}" style="max-width: 400px;">
                    <div class="help-text">Default API token to pre-fill when adding new DNS records (can be changed per record)</div>
                </div>
            </div>
            
            <div class="settings-section">
                <h3>🌐 Web Interface Settings</h3>
                
                <div class="form-group">
                    <label>Web Interface Port:</label>
                    <input type="number" name="web_port" value="{{.Config.Web.Port}}" min="1" max="65535">
                    <div class="help-text">Port for the web interface (requires restart to take effect)</div>
                </div>
            </div>
            
            <button type="submit" class="btn btn-primary">Save Settings</button>
            <a href="/" class="btn btn-secondary">Cancel</a>
        </form>
    </div>
</body>
</html>`
