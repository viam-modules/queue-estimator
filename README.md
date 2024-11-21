# queue-estimator
models that summarize information from underlying vision models for waiting

https://app.viam.com/module/viam/queue-estimator

Using an underlying camera and vision service, you can use information from the vision service to measure crowding around an area of interest.

You specify the area of interset by filling out the cropping box attribute in the config. If the cropping box is empty, the sensor will use the entire scene.

Rather than just counting how many people are there in the area of interest (which could jump wildly up and down as people pass) to determine whether there a wait time, the algo instead counts to see if enough people are in that area of interest (as determined by the trigger_threshold) then increments a trigger_count as long as there are at least that many people in that area. This is supposed to capture the fact that many people will be congregating in the area, and not move away from it.

- The trigger_count increments when the number of people is at or above the trigger_threshold
- The trigger_count decrements when the number of people falls below the trigger_threshold
- The label of occupancy (or wait times) associated with the location is determined by how high the trigger_count is
- The trigger_count cannot go below 0, or above the largest upper bound set by the count_thresholds attribute. 

## Example Config

### wait-sensor
- trigger_threshold: an int that represents how many people should be in the scene before the trigger count starts incrementing. numbers below the trigger will not make the count increase.
- count_threshold: as the trigger count increases, a corresponding string will be associated with the count. the value is the upper-bound of the trigger count. Anything below this number will be give the associated string label.
- detector_name: the underlying vision service detector to use
- camera_name: the underlying camera the vision service detector should use
- poll_frequency_hz: how often to poll the underlying vision service, in Hz
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
  "trigger_threshold": 4 # requires at least 4 detections before count goes up
  "cropping_box": [0.3, 0.33, 0.6. 0.65],
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
