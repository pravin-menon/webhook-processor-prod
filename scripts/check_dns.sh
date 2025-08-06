while true; do
    echo "
Checking DNS propagation for webhook-dev.wizrdom.com..."
    dig webhook-dev.wizrdom.com +short
    if [ $? -eq 0 ] && [ -n "$(dig webhook-dev.wizrdom.com +short)" ]; then
        echo "
✅ DNS record is active!"
        echo "Cloudflare IPs found:"
        dig webhook-dev.wizrdom.com +short
        break
    fi
    echo "Still waiting for DNS propagation... (checking again in 30 seconds)"
    sleep 30
done
