# queue-estimator
models that summarize information from underlying vision models for waiting

https://app.viam.com/module/viam/queue-estimator

Using an underlying set of cameras and vision service, you can use information from the vision service to measure crowding around areas of interest.

You specify the areas of interset by filling out the `valid_regions` attribute in the config. If the cropping box list for a camera is empty, the sensor will use the entire scene from that camera.

Rather than just counting the sum of people in the areas of interest (which could jump wildly up and down as people pass) to determine whether there a wait time, the algo instead counts to see if enough people are in those areas of interest over a period of time (as determined by the sampling_period_s) and then updates the state of the queue as appropriate. The levels of crowdedness are determined by by the "count_thresholds", as well as how many samples (n_samples) have been taken within the counting period.

The level of crowdedness is updated continously usually a rolling average based on the past n_samples from the regions of the interest.
## Example Config

### wait-sensor
- sampling_period_s: this is how long to "look back" for in order to make a decision based on average crowdedness over this time period.
- n_samples: how often to poll the underlying vision service within the sampling_period_s.
- valid_regions: the underlying cameras and regions of interest within the respective camera scenes the vision service detector should use. The minimum and maximum X and Y values should each be between 0 and 1.
- count_threshold: the corresponding string associated with the count for one sample. the value is the upper-bound of the trigger count. Anything below this number will be give the associated string label.
- detector_name: the underlying vision service detector to use
- chosen_labels: what are the labels  and confidence scores of the underlying vision service that should count towards the count.
- extra_fields: any extra fields that should be copied to the sensor output
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
