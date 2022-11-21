# Summary
Sometimes you need to run nginx ingress controller not directly behind NLB but behind ALB. Maybe you need to use Global Accelerator or some other integrated service. At some time, you cannot use ALB directly because you need some features not offered by ALB, like messing with headers or rewriting routes. 

To achieve that you can always deploy Nginx ingress controller with NodePort service mode and in ALB Load Balancer Controller ingress, add catch-all rule to pass all traffic to nginx. There is but one issue. If you use ExternalDNS to manage DNS records, you need to configure nginx ingress controller with publishedService that reflects ALB DNS name in service. Since ALB Load Balancer Controller creates ALB in AWS resources directly, only reference to its DNS is in status of ALB ingress object. 

Catch is that Nginx ingress controller can only use Service as source of status record for its ingress objects, not another ingress. This is where this little service helps. It watches given ALB ingress and copies its dns name to  Service that is of type ExternaName and has annotation: 

```  
  tickmill.com/nginx.frontrunner: <name of alb ingress>
```

With this workaround, any ingress that is created for nginx ingress controller to manage, will have status hostname set to value of ALB dns. ExternalDNS uses this to manage CNAME records with correct target and you'll be happy. 
