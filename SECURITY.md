# Security Policy

## Supported Versions

Gorilix is currently in early development. The following versions are currently being maintained with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

We take the security of Gorilix seriously. If you believe you've found a security vulnerability, please follow these steps:

1. **Do Not** disclose the vulnerability publicly (no GitHub issues, pull requests, or social media posts)
2. Email the maintainers directly at [me@juliaklee.wtf] (replace with actual contact once established)
3. Include as much information as possible:
   - A description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix, if possible

## What to Expect

After reporting a vulnerability:

1. You'll receive an acknowledgment within 48 hours
2. We'll investigate and provide an assessment within 7 days
3. We'll work with you to understand and address the issue
4. Once resolved, we'll publish the fix and credit you (unless you prefer to remain anonymous)

## Security Best Practices

When using Gorilix in production environments:

1. Always use the latest version with security updates
2. Follow secure coding practices when implementing actors and message handlers
3. Implement appropriate authentication and authorization in your application
4. Consider network security when deploying across multiple machines

## Security Features

Gorilix's actor model inherently provides isolation between components, which can help limit the impact of security issues. The circuit breaker pattern can also prevent cascading failures due to security-related problems.

However, Gorilix is not a security framework itself and should be used alongside proper security controls for your specific use case. 