# `queue-estimator` module

This module implements the [Viam sensor API](https://docs.viam.com/dev/reference/apis/components/sensor/) to monitor queue wait times and occupancy levels using vision detection. Instead of simple person counting, it analyzes occupancy patterns over time to provide stable status updates.

## Model: `viam:queue-estimator:wait-sensor`

The wait-sensor monitors areas of interest using vision detection and provides status updates based on rolling averages of person counts over a configurable time period.

### Configuration

The following attributes are available for this model:

| Name | Type | Inclusion | Description |
| ---- | ---- | --------- | ----------- |
| `n_samples` | int | **Required** | Number of detection samples to collect evenly spaced over the sampling period. Higher values = smoother averaging. |
| `valid_regions` | object | **Required** | Maps camera names to lists of bounding box regions `{"x_min": 0-1, "y_min": 0-1, "x_max": 0-1, "y_max": 0-1}`. Use `[{}]` for entire camera view. |
| `count_thresholds` | object | **Required** | Maps custom status labels to minimum person count thresholds. Example: `{"No wait": 0, "Short wait": 2, "Long wait": 5}` |
| `detector_name` | string | **Required** | Name of your vision service detector. |
| `chosen_labels` | object | **Required** | Detection labels and confidence thresholds to count. Example: `{"person": 0.6}` |
| `sampling_period_s` | float | Optional | Time window in seconds for rolling average. Default: 30 |
| `extra_fields` | object | Optional | Additional metadata to include in sensor output. |

### Example Configuration

```json
{
  "n_samples": 5,
  "valid_regions": {
    "main_camera": [{}]
  },
  "count_thresholds": {
    "No wait": 0,
    "Short wait": 2,
    "Medium wait": 5,
    "Long wait": 10,
    "Very long wait": 1000
  },
  "detector_name": "person_detector",
  "chosen_labels": {
    "person": 0.6
  }
}
```

### Multi-Camera Example

```json
{
  "sampling_period_s": 10,
  "n_samples": 5,
  "valid_regions": {
    "entrance_camera": [
      {
        "x_min": 0.3,
        "y_min": 0.3,
        "x_max": 0.7,
        "y_max": 0.7
      }
    ],
    "exit_camera": [{}]
  },
  "count_thresholds": {
    "green": 2,
    "yellow": 5,
    "red": 10
  },
  "detector_name": "vision-1",
  "chosen_labels": {
    "person": 0.7
  },
  "extra_fields": {
    "location_name": "Main Entrance"
  }
}
```

### How count_thresholds Work

Define custom status labels with minimum person counts:
- Status returned when count is >= threshold but < next threshold
- Use any naming: `"No wait"`, `"green"`, `"0_min"`, etc.
- Example: With thresholds `{"green": 0, "yellow": 3, "red": 8}`:
  - 0-2 people → returns "green"
  - 3-7 people → returns "yellow"  
  - 8+ people → returns "red"

### Configuration Tips

- **Valid regions:** Use `[{}]` for full camera view, or specify bounding boxes for specific areas
- **Confidence:** Start with 0.6-0.7; lower (0.4-0.5) for poor lighting, higher (0.7-0.8) to reduce false positives
- **Samples:** More samples (10-15) = smoother but slower response; fewer (3-5) = faster updates
- **Thresholds:** Leave gaps between values to prevent status flipping