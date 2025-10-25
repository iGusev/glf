# Security Policy

## Supported Versions

We actively support the following versions of GLF with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 0.3.x   | :white_check_mark: |
| < 0.3   | :x:                |

We recommend always using the latest release to ensure you have the most up-to-date security patches.

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue in GLF, please report it responsibly.

### How to Report

**Please do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please email security reports to: **[mail@igusev.ru](mailto:mail@igusev.ru)**

Include the following information in your report:

- **Description** of the vulnerability
- **Steps to reproduce** the issue
- **Potential impact** of the vulnerability
- **Suggested fix** (if you have one)
- Your contact information for follow-up questions

### What to Expect

- **Acknowledgment**: We will acknowledge receipt of your report within 48 hours
- **Updates**: We will provide regular updates on our progress
- **Timeline**: We aim to address critical vulnerabilities within 7 days
- **Credit**: With your permission, we will credit you in the release notes

### Disclosure Policy

- Please allow us reasonable time to address the issue before public disclosure
- We will coordinate with you on the disclosure timeline
- We will publicly acknowledge your responsible disclosure (unless you prefer to remain anonymous)

### Security Best Practices

When using GLF, we recommend:

1. **Protect your GitLab token**:
   - Never commit tokens to version control
   - Use restrictive file permissions for `~/.config/glf/config.yaml` (0600)
   - Regularly rotate your GitLab access tokens

2. **Keep GLF updated**:
   - Use the latest stable version
   - Subscribe to release notifications on GitHub

3. **Secure your environment**:
   - Use HTTPS for GitLab connections
   - Ensure your GitLab instance is properly secured

## Security Updates

Security updates will be released as patch versions and announced via:

- GitHub Security Advisories
- Release notes on GitHub
- Project README

Thank you for helping keep GLF and its users safe!
