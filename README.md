# queue-estimator
models that summarize information from underlying vision models for waiting

https://app.viam.com/module/viam/queue-estimator

Using an underlying set of cameras and vision service, you can use information from the vision service to measure crowding around areas of interest.

You specify the areas of interset by filling out the `valid_regions` attribute in the config. If the cropping box list for a camera is empty, the sensor will use the entire scene from that camera.

Rather than just counting the sum of people in the areas of interest (which could jump wildly up and down as people pass) to determine whether there a wait time, the algo instead counts to see if enough people are in those areas of interest over a period of time (as determined by the sampling_period_s) and then updates the state of the queue as appropriate. The levels of crowdedness are determined by by the "count_thresholds", as well as how many samples (n_samples) have been taken within the counting period.

The level of crowdedness is updated continously usually a rolling average based on the past n_samples from the regions of the interest.

## Attributes

| Name | Type | Inclusion | Description |
| ---- | ---- | --------- | ----------- |
| `sampling_period_s` | float64 | Optional | We "look back" over this amount of time to average the crowdedness. The default is 30 seconds. |
| `n_samples` | int | Required | How many images to take, evenly spaced over `sampling_period_s` seconds. |
| `valid_regions` | map | Required | Maps camera names (strings) to lists of objects describing bounding boxes within that camera image to search. Use an empty object to examine the entire image.  Otherwise, each object should contain the fields `x_min`, `x_max`, `y_min`, and `y_max`. All four fields should be floats between 0 and 1 and represent a fraction of the image's entire width/height. |
| `count_thresholds` | map | Required | Maps labels the sensor should return (a string) to the maximum number of people detected when triggering this label (an int). Anything at or below this value (but higher than all lower values) will return this label. |
| `detector_name` | string | Required | The underlying vision service detector to use |
| `chosen_labels` | map | Required | The labels and minimum confidence scores of the underlying vision service that should be included in the total count. |
| `extra_fields` | object | Optional | Any extra fields that should be copied to the sensor output |

## Example Config

### wait-sensor
```
"name": "queue-sensor",
"namespace": "rdk",
"type": "sensor",
"model": "viam:queue-estimator:wait-sensor",
"attributes": {
  "count_thresholds": {
    "0_min": 3,
    "2_min": 7,
    "7_min": 14,
    "10_min": 20,
    ">10_min": 30
  },
  "sampling_period_s": 10,
  "n_samples": 5, # will measure the crowd every 2 seconds
  "valid_regions": {
     "camera_3": [{"x_min": 0.3,
                   "y_min": 0.33,
                   "x_max": 0.6,
                   "y_max": 0.65}],
     "camera_12": [{"x_min": 0.2,
                    "y_min": 0.1,
                    "x_max": 0.3,
                    "y_max": 0.3},
                   {"x_min": 0.75,
                    "y_min": 0.75,
                    "x_max": 1.0,
                    "y_max": 1.0}],
     "camera_44": [], # this means use the whole camera scene
  },
  "detector_name": "vision-1",
  "chosen_labels": {
    "person": 0.3
  },
  "extra_fields": {
    "location_open": true,
    "location_name": "store_2"
  }
}
```
