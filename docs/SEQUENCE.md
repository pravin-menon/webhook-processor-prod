sequenceDiagram
    participant MC as MailerCloud
    participant LB as Load Balancer
    participant WH as Webhook Handler
    participant RL as Rate Limiter
    participant RMQ as RabbitMQ
    participant WP as Worker Processor
    participant DB as MongoDB
    participant Mon as Monitoring

    MC->>LB: POST /webhook
    LB->>WH: Route Request
    WH->>WH: Validate Headers
    WH->>RL: Check Rate Limits
    
    alt Rate Limit Exceeded
        RL-->>WH: Reject Request
        WH-->>MC: 429 Too Many Requests
    else Rate Limit OK
        RL-->>WH: Accept Request
        WH->>RMQ: Publish Event
        RMQ-->>WH: Confirm Receipt
        WH-->>MC: 202 Accepted
        
        loop Until Processed
            RMQ->>WP: Consume Event
            WP->>DB: Store Event
            
            alt Process Success
                DB-->>WP: Confirm Storage
                WP->>Mon: Record Success
            else Process Failed
                DB-->>WP: Report Error
                WP->>RMQ: Retry (with backoff)
                WP->>Mon: Record Failure
            end
        end
    end

    Mon->>Mon: Update Metrics
    Mon->>Mon: Check Thresholds
    
    alt Threshold Exceeded
        Mon->>WP: Generate Alert
    end
