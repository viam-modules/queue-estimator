# queue-estimator
models that summarize information from underlying vision models for waiting

https://app.viam.com/module/viam/queue-estimator

## Example Config

### wait-sensor
- count_threshold: the value is the upper-bound. Anything below this number will be give the associated string label.
- detector_name: the underlying vision service detector to use
- camera_name: the underlying camera the vision service detector should use
- poll_frequency_hz: how often to poll the underlying vision service, in Hz
- chosen_labels: what are the labels  and confidence scores of the underlying vision service that should count towards the count.
- extra_filds: any extra fields that should be copied to the sensor output
```
"name": "queue-sensor",
"namespace": "rdk",
"type": "sensor",
"model": "viam:queue-estimator:wait-sensor",
"attributes": {
  "count_thresholds": {
    ">10_min": 1000,
    "0_min": 2,
    "2_min": 7,
    "7_min": 14,
    "10_min": 20,
  },
  "detector_name": "vision-1",
  "camera_name": "camera-1",
  "poll_frequency_hz": 0.5,
  "chosen_labels": {
    "person": 0.3
  },
  "extra_fields": {
    "location_open": true,
    "location_name": "store_2"
  }
}
```
