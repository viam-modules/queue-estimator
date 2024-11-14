# queue-estimator
models that summarize information from underlying vision models for waiting

## Example Config

### wait count-sensor
```
{
  "count_thresholds": {
    ">10_min": 1000,
    "0_min": 3,
    "2min": 10,
    "7min": 20
  },
  "detector_name": "vision-1",
  "camera_name": "camera-1",
  "poll_frequency_hz": 0.5,
  "chosen_labels": {
    "person": 0.3
  }
}
```
