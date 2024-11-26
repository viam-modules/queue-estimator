# queue-estimator
models that summarize information from underlying vision models for waiting

https://app.viam.com/module/viam/queue-estimator

Using an underlying camera and vision service, you can use information from the vision service to measure crowding around an area of interest.

You specify the area of interset by filling out the cropping box attribute in the config. If the cropping box is empty, the sensor will use the entire scene.

Rather than just counting how many people are there in the area of interest (which could jump wildly up and down as people pass) to determine whether there a wait time, the algo instead counts to see if enough people are in that area of interest over a period of time (as determined by the sampling_period_s) and then updates the state of the queue as appropriate. The levels of crowdedness are determined by by the "count_thresholds", as well as how many samples (n_samples) have been taken within the counting period.

## Example Config

### wait-sensor
- sampling_period_s: this is how long to gather data before updating the state of the queue estimator.
- n_samples: how often to poll the underlying vision service within the sampling_period_s. 
- count_threshold: the corresponding string associated with the count. the value is the upper-bound of the trigger count. Anything below this number will be give the associated string label.
- detector_name: the underlying vision service detector to use
- camera_name: the underlying camera the vision service detector should use
- chosen_labels: what are the labels  and confidence scores of the underlying vision service that should count towards the count.
- extra_fields: any extra fields that should be copied to the sensor output
- cropping_box: `[x_min, y_min, x_max, y_max]` to crop the image to, and only count objects within that box. the box is using relative scale to the image dimension, e.g. `[0.3, .0.25, 0.8, 0.68]`
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
  "cropping_box": [0.3, 0.33, 0.6. 0.65],
  "detector_name": "vision-1",
  "camera_name": "camera-1",
  "chosen_labels": {
    "person": 0.3
  },
  "extra_fields": {
    "location_open": true,
    "location_name": "store_2"
  }
}
```
