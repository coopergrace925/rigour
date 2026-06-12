package api

import (
	"html/template"
	"net/http"
)

const scanInfoHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Rigour Scanner Information</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Courier New', monospace;
            background: #0a0a0a;
            color: #ffa500;
            line-height: 1.6;
            padding: 2rem;
        }
        .container {
            max-width: 900px;
            margin: 0 auto;
            border: 2px solid #ffa500;
            padding: 2rem;
            box-shadow: 0 0 20px rgba(255, 165, 0, 0.3);
        }
        h1 {
            color: #ffa500;
            border-bottom: 2px solid #ffa500;
            padding-bottom: 1rem;
            margin-bottom: 2rem;
            text-transform: uppercase;
            letter-spacing: 2px;
        }
        h2 {
            color: #ffa500;
            margin-top: 2rem;
            margin-bottom: 1rem;
            font-size: 1.3rem;
        }
        p, li {
            margin-bottom: 1rem;
            color: #d4d4d4;
        }
        a {
            color: #ffa500;
            text-decoration: none;
            border-bottom: 1px solid #ffa500;
        }
        a:hover {
            background: #ffa500;
            color: #0a0a0a;
        }
        .highlight {
            background: #1a1a1a;
            padding: 1rem;
            border-left: 4px solid #ffa500;
            margin: 1rem 0;
        }
        ul {
            list-style: none;
            padding-left: 1rem;
        }
        ul li:before {
            content: "▸ ";
            color: #ffa500;
            font-weight: bold;
        }
        code {
            background: #1a1a1a;
            padding: 0.2rem 0.5rem;
            color: #ffa500;
            border: 1px solid #333;
        }
        .api-box {
            background: #1a1a1a;
            border: 1px solid #ffa500;
            padding: 1.5rem;
            margin: 1rem 0;
        }
        .footer {
            margin-top: 3rem;
            padding-top: 2rem;
            border-top: 1px solid #ffa500;
            text-align: center;
            font-size: 0.9rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🔍 Rigour Internet Scanner</h1>
        
        <div class="highlight">
            <p><strong>You are receiving traffic from this scanner because we are conducting research on internet-wide security and service availability.</strong></p>
        </div>

        <h2>About Rigour</h2>
        <p>Rigour is a <strong>world-class internet scanner</strong> designed for security research, vulnerability assessment, and internet measurement. We scan <strong>3,847 ports</strong> across the public IPv4 address space to help organizations understand their internet exposure.</p>

        <h2>What We Scan</h2>
        <ul>
            <li><strong>3,847 ports</strong> using Shodan's industry-standard port list</li>
            <li><strong>Banner collection only</strong> - No exploitation or intrusion</li>
            <li><strong>Protocol detection</strong> - SSH, HTTP, FTP, databases, ICS/SCADA</li>
            <li><strong>TLS certificate extraction</strong> - For domain attribution</li>
            <li><strong>Reverse DNS lookup</strong> - For hostname identification</li>
        </ul>

        <h2>Scan Frequency</h2>
        <p>We use adaptive scheduling based on port criticality:</p>
        <ul>
            <li><strong>Critical ports (SSH, RDP, databases):</strong> Every 6 hours</li>
            <li><strong>Web services (HTTP/HTTPS):</strong> Every 24 hours</li>
            <li><strong>ICS/SCADA protocols:</strong> Every 24 hours</li>
            <li><strong>Common ports:</strong> Every 7 days</li>
            <li><strong>Full port list:</strong> Every 30 days</li>
        </ul>

        <h2>Our Commitment</h2>
        <ul>
            <li><strong>Non-intrusive:</strong> Banner grabbing only, no exploitation</li>
            <li><strong>Rate limited:</strong> 100 scans/minute per ASN (configurable)</li>
            <li><strong>Respectful:</strong> Honor robots.txt and opt-out requests</li>
            <li><strong>Transparent:</strong> Full technical documentation available</li>
            <li><strong>Ethical:</strong> Research purposes only, no malicious intent</li>
        </ul>

        <h2>Opt-Out Instructions</h2>
        <p>If you wish to exclude your IP address or network from our scans, we honor opt-out requests:</p>
        
        <div class="api-box">
            <p><strong>Opt-Out API Endpoint:</strong></p>
            <code>POST {{.BaseURL}}/api/opt-out</code>
            <p style="margin-top: 1rem;"><strong>Request Body (JSON):</strong></p>
            <pre style="color: #d4d4d4; margin-top: 0.5rem;">{
  "ip_or_cidr": "203.0.113.0/24",
  "email": "admin@example.com",
  "reason": "Production network"
}</pre>
            <p style="margin-top: 1rem;"><strong>Or email us directly:</strong> <a href="mailto:{{.ContactEmail}}">{{.ContactEmail}}</a></p>
        </div>

        <p><strong>Processing time:</strong> Opt-outs are processed within 24 hours.</p>

        <h2>Contact Information</h2>
        <ul>
            <li><strong>Email:</strong> <a href="mailto:{{.ContactEmail}}">{{.ContactEmail}}</a></li>
            <li><strong>Security.txt:</strong> <a href="/.well-known/security.txt">/.well-known/security.txt</a></li>
            <li><strong>GitHub:</strong> <a href="{{.GitHubURL}}" target="_blank">{{.GitHubURL}}</a></li>
            <li><strong>Website:</strong> <a href="{{.WebsiteURL}}" target="_blank">{{.WebsiteURL}}</a></li>
        </ul>

        <h2>Technical Details</h2>
        <ul>
            <li><strong>Scanner Name:</strong> Rigour</li>
            <li><strong>User-Agent:</strong> Rigour/2.0 (+{{.BaseURL}}/scaninfo)</li>
            <li><strong>Technology:</strong> ZMap + ZGrab2 (industry standard)</li>
            <li><strong>Port Coverage:</strong> 3,847 ports (Shodan list)</li>
            <li><strong>Architecture:</strong> World-class, horizontally scalable</li>
        </ul>

        <h2>Research Purpose</h2>
        <p>Our scanning activities support:</p>
        <ul>
            <li><strong>Security research:</strong> Vulnerability assessment and internet measurement</li>
            <li><strong>Academic studies:</strong> Internet topology and service distribution</li>
            <li><strong>Threat intelligence:</strong> Early detection of compromised systems</li>
            <li><strong>Open source contribution:</strong> Public internet security dataset</li>
        </ul>

        <h2>Legal Compliance</h2>
        <p>Rigour operates in accordance with applicable laws and industry best practices:</p>
        <ul>
            <li>RFC 1918, DoD, and IANA reserved IPs are excluded</li>
            <li>Opt-out requests are honored promptly</li>
            <li>No exploitation or unauthorized access attempts</li>
            <li>Banner collection is considered legal research</li>
        </ul>

        <div class="footer">
            <p>Rigour Internet Scanner | World-Class Security Research Platform</p>
            <p>Last Updated: 2026-06-12 | <a href="{{.GitHubURL}}">Open Source on GitHub</a></p>
        </div>
    </div>
</body>
</html>`

// ScanInfoConfig holds configuration for the scan info page
type ScanInfoConfig struct {
	BaseURL      string
	ContactEmail string
	GitHubURL    string
	WebsiteURL   string
}

// HandleScanInfo serves the scan information page
func HandleScanInfo(config ScanInfoConfig) http.HandlerFunc {
	tmpl := template.Must(template.New("scaninfo").Parse(scanInfoHTML))
	
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, config); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// HandleSecurityTxt serves the security.txt file (RFC 9116)
func HandleSecurityTxt(config ScanInfoConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		
		securityTxt := `# security.txt for Rigour Internet Scanner
# Rigour is a world-class internet scanner for security research
# Last Updated: 2026-06-12

Contact: mailto:` + config.ContactEmail + `
Expires: 2027-12-31T23:59:59Z
Preferred-Languages: en
Canonical: ` + config.BaseURL + `/.well-known/security.txt

# Scanner Information
# Name: Rigour
# Purpose: Security research and internet measurement
# Scaninfo: ` + config.BaseURL + `/scaninfo
# Opt-out: ` + config.BaseURL + `/api/opt-out

# About Rigour
# Rigour scans 3,847 ports using industry-standard tools (ZMap + ZGrab2)
# We perform banner collection only - no exploitation or intrusion
# We honor opt-out requests within 24 hours

# Rate Limiting
# Default: 100 scans/minute per ASN (configurable)
# Critical ports: Every 6 hours
# Web services: Every 24 hours
# Full port list: Every 30 days

# Contact for:
# - Opt-out requests
# - Security concerns
# - Research collaboration
# - Technical questions

Acknowledgments: ` + config.GitHubURL + `
`
		w.Write([]byte(securityTxt))
	}
}
