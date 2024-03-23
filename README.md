# Ritual
An AI-powered command line application for maintaining rituals through activity logging and accountability.

# Install
Clone, build, run.
```bash
git clone https://github.com/JTan2231/ritual.git
cd ritual
go build
./ritual
```

# Usage
Before you get started, make an account and save your email/password in `$RITUAL_USERNAME` and `$RITUAL_PASSWORD` environment variables:
```bash
./ritual signup your.email@emailprovider.com your_password
export RITUAL_USERNAME=your.email@emailprovider.com
export RITUAL_PASSWORD=your_password
```

Activities are the core of Ritual. Every time you finish an activity (e.g., programming), record it to Ritual like so:
```bash
# ./ritual log <activity_name> <duration (minutes)> <memo>
./ritual log walk 60 "daily stroll"
```
On recording your activity, you'll receive some words of wisdom and feedback from GPT based on its consistency with some of your recently recorded activities:
![image](https://github.com/JTan2231/ritual/assets/37962780/51645f39-299e-4569-8405-77b3eedf1f89)
