# queue-estimator
models that summarize information from underlying vision models for waiting

## Example Config

### wait-sensor
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
