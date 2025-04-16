package waitsensor

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/vision"
	"go.viam.com/rdk/utils"
	objdet "go.viam.com/rdk/vision/objectdetection"
	viamutils "go.viam.com/utils"
)

const (
	// ModelName is the name of the model
	ModelName = "wait-sensor"
	// OverflowLabel is the label if the counts exceed what was specified by the user
	OverflowLabel = "Overflow"
	// DefaultMaxFrequency is how often the vision service will poll the camera for a new image
	DefaultPollFrequency = 1.0
	// DefaultTransitionTime is how long it takes for the state to change by default
	DefaultTransitionTime = 30.0
)

var (
	// Model is the resource
	Model = resource.NewModel("viam", "queue-estimator", ModelName)
)

func init() {
	resource.RegisterComponent(sensor.API, Model, resource.Registration[sensor.Sensor, *Config]{
		Constructor: newWaitSensor,
	})
}

type BoundingBoxConfig struct {
	XMin float64 `json:"x_min"`
	XMax float64 `json:"x_max"`
	YMin float64 `json:"y_min"`
	YMax float64 `json:"y_max"`
}

// Config contains names for necessary resources
type Config struct {
	DetectorName    string                         `json:"detector_name"`
	ChosenLabels    map[string]float64             `json:"chosen_labels"`
	CountThresholds map[string]int                 `json:"count_thresholds"`
	CountPeriod     float64                        `json:"sampling_period_s"`
	NSamples        int                            `json:"n_samples"`
	ExtraFields     map[string]interface{}         `json:"extra_fields"`
	ValidRegions    map[string][]BoundingBoxConfig `json:"valid_regions"`
}

// Validate validates the config and returns implicit dependencies,
// this Validate checks if the camera and detector exist for the module's vision model.
func (cfg *Config) Validate(path string) ([]string, error) {
	camAndDetNames := []string{}
	if cfg.DetectorName == "" {
		return nil, errors.New("attribute detector_name cannot be left blank")
	}
	camAndDetNames = append(camAndDetNames, cfg.DetectorName)
	if len(cfg.ValidRegions) == 0 {
		return nil, errors.New("attribute valid_regions cannot be left blank")
	}
	if len(cfg.CountThresholds) == 0 {
		return nil, errors.New("attribute count_thresholds is required")
	}
	if cfg.NSamples <= 0 {
		return nil, errors.New("attribute n_samples must be greater than 0")
	}
	if cfg.CountPeriod < 0 {
		return nil, errors.New("attribute sampling_period_s cannot be less than 0. default is 30s")
	}
	testMap := map[int]string{}
	for label, v := range cfg.CountThresholds {
		if _, ok := testMap[v]; ok {
			return nil, errors.Errorf("cannot have two labels for the same threshold in count_thresholds. Threshold value %v appears more than once", v)
		}
		testMap[v] = label
	}
	for camName, listBB := range cfg.ValidRegions {
		camAndDetNames = append(camAndDetNames, camName)
		for _, coords := range listBB {
			_, err := NewBoundingBox(coords)
			if err != nil {
				return nil, errors.Wrapf(err, "error in valid_region for %v", camName)
			}
		}
	}
	return camAndDetNames, nil
}

// Bin stores the thresholds that turns counts into labels
type Bin struct {
	UpperBound float64
	Label      string
}

// NewThresholds creates a list of thresholds for labeling counts
func NewThresholds(t map[string]int) []Bin {
	// first invert the map, Validate ensures a 1-1 mapping
	thresholds := map[int]string{}
	for label, val := range t {
		thresholds[val] = label
	}
	out := []Bin{}
	keys := []int{}
	for k := range thresholds {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	for _, key := range keys {
		b := Bin{float64(key), thresholds[key]}
		out = append(out, b)
	}
	return out
}

type BoundingBox struct {
	xMin, yMin, xMax, yMax float64
}

func NewBoundingBox(coords BoundingBoxConfig) (BoundingBox, error) {
	for _, e := range []float64{coords.XMin, coords.YMin, coords.XMax, coords.YMax} {
		if e < 0.0 || e > 1.0 {
			return BoundingBox{}, errors.New("bounding box numbers are relative to the image dimension, and must be numbers between 0 and 1.")
		}
	}
	// If our "bounding box" is the whole image, XMax and/or YMax will be 0, in which case it can
	// be equal to XMin/YMin. Otherwise, make sure the max is larger than the min.
	if coords.XMax > 0 && coords.XMin >= coords.XMax {
		return BoundingBox{}, fmt.Errorf(
			"x_min (%f) must be less than x_max (%f)", coords.XMin, coords.XMax)
	}
	if coords.YMax > 0 && coords.YMin >= coords.YMax {
		return BoundingBox{}, fmt.Errorf(
			"y_min (%f) must be less than y_max (%f)", coords.YMin, coords.YMax)
	}
	bb := BoundingBox{xMin: coords.XMin,
	                  yMin: coords.YMin,
	                  xMax: coords.XMax,
	                  yMax: coords.YMax,
	                  }
	return bb, nil
}

// crop coordinates were already validated upon configuration
// empty bounding box implies no crop
func (bb BoundingBox) Crop(img image.Image) image.Image {
	if bb.xMax == 0 || bb.yMax == 0 {
		return img
	}
	// Get image bounds
	bounds := img.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	// Convert relative coordinates to absolute pixels
	x1 := bounds.Min.X + int(bb.xMin*float64(width))
	y1 := bounds.Min.Y + int(bb.yMin*float64(height))
	x2 := bounds.Min.X + int(bb.xMax*float64(width))
	y2 := bounds.Min.Y + int(bb.yMax*float64(height))

	// Create cropping rectangle
	rect := image.Rect(x1, y1, x2, y2)
	croppedImg := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(croppedImg, croppedImg.Bounds(), img, rect.Min, draw.Src)
	return croppedImg
}

type syncValues struct {
	mu              sync.RWMutex
	numInView       int
	class           string
	mean            float64
	countListString string
}

type counter struct {
	resource.Named
	cancelFunc              context.CancelFunc
	cancelContext           context.Context
	activeBackgroundWorkers sync.WaitGroup
	logger                  logging.Logger
	detName                 string
	detector                vision.Service
	cams                    map[string]camera.Camera
	validRegions            map[string][]BoundingBox
	labels                  map[string]float64
	thresholds              []Bin
	frequency               float64
	extraFields             map[string]interface{}
	transitionCount         int
	syncVals                *syncValues
}

func newWaitSensor(
	ctx context.Context,
	deps resource.Dependencies,
	conf resource.Config,
	logger logging.Logger) (sensor.Sensor, error) {
	cs := &counter{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}

	if err := cs.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}
	return cs, nil
}

// Reconfigure resets the underlying detector as well as the thresholds and labels for the count
func (cs *counter) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	if cs.cancelFunc != nil {
		cs.cancelFunc()
		cs.activeBackgroundWorkers.Wait()
	}
	cancelableCtx, cancel := context.WithCancel(context.Background())
	cs.cancelFunc = cancel
	cs.cancelContext = cancelableCtx

	cs.syncVals = &syncValues{}
	countConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return errors.Errorf("Could not assert proper config for %s", ModelName)
	}
	cs.extraFields = map[string]interface{}{}
	if countConf.ExtraFields != nil {
		cs.extraFields = countConf.ExtraFields
	}
	cs.validRegions = make(map[string][]BoundingBox)
	cs.cams = make(map[string]camera.Camera)
	for camName, bbList := range countConf.ValidRegions {
		// first store the cameras from dependencies
		cn := camName
		cam, err := camera.FromDependencies(deps, cn)
		if err != nil {
			return errors.Wrapf(err, "unable to get camera %v for count classifier", cn)
		}
		cs.cams[cn] = cam
		// next store the valid regions as a list of bounding boxes
		cs.validRegions[cn] = []BoundingBox{}
		for _, coords := range bbList {
			bb, err := NewBoundingBox(coords)
			if err != nil {
				return errors.Wrapf(err, "error in valid region for %v", cn)
			}
			cs.validRegions[cn] = append(cs.validRegions[cn], bb)
		}
	}
	// transition time in seconds
	transitionTime := DefaultTransitionTime
	if countConf.CountPeriod > 0 {
		transitionTime = countConf.CountPeriod
	}
	cs.transitionCount = countConf.NSamples
	cs.frequency = transitionTime / float64(cs.transitionCount)
	cs.logger.Infof("number of samples/time between state change for queue estimator, n_samples: %v, time (s): %v. number of cameras: %v", cs.transitionCount, transitionTime, len(cs.cams))
	cs.detName = countConf.DetectorName
	cs.detector, err = vision.FromDependencies(deps, countConf.DetectorName)
	if err != nil {
		return errors.Wrapf(err, "unable to get vision service %v for count classifier", countConf.DetectorName)
	}
	// put everything in lower case
	labels := map[string]float64{}
	for l, c := range countConf.ChosenLabels {
		labels[strings.ToLower(l)] = c
	}
	cs.labels = labels
	cs.thresholds = NewThresholds(countConf.CountThresholds)
	// now start the background thread
	cs.activeBackgroundWorkers.Add(1)
	viamutils.ManagedGo(func() {
		// if you get an error while running just keep trying forever
		for {
			runErr := cs.run(cs.cancelContext)
			if runErr != nil {
				if strings.Contains(runErr.Error(), "context canceled") {
					return
				}
				cs.logger.Errorw("background thread exited with error", "error", runErr)
				continue // keep trying to run, forever
			}
			return
		}
	}, func() {
		cs.activeBackgroundWorkers.Done()
	})
	return nil
}

func (cs *counter) countDets(dets []objdet.Detection) int {
	// get the number of boxes with the right label and confidences
	count := 0
	for _, d := range dets {
		label := strings.ToLower(d.Label())
		if conf, ok := cs.labels[label]; ok {
			if d.Score() >= conf {
				count++
			}
		}
	}
	return count
}

// return the level, and the string associated with that level
func counts2class(count float64, thresholds []Bin) string {
	// associated the number with the right label
	for _, thresh := range thresholds {
		if count <= thresh.UpperBound {
			return thresh.Label
		}
	}
	return OverflowLabel
}

type RingBuffer struct {
	size    int
	data    []float64
	current int
	full    bool
}

func NewRingBuffer(size int) *RingBuffer {
	data := make([]float64, size)
	return &RingBuffer{
		size: size,
		data: data,
	}
}

func (rb *RingBuffer) Add(newCount float64) {
	rb.data[rb.current] = newCount
	rb.current = (rb.current + 1) % rb.size
	if rb.current == 0 { // the first time it reaches this again, it will be full
		rb.full = true
	}
}

func (rb *RingBuffer) Len() int {
	if rb.full {
		return rb.size
	}
	return rb.current
}

func (rb *RingBuffer) Mean() float64 {
	num := rb.Len()
	if num == 0 {
		return 0.0
	}
	avg := 0.0
	for _, v := range rb.data {
		avg += v
	}
	return avg / float64(num)
}

func (rb *RingBuffer) MeanLabel(thresholds []Bin) string {
	avg := rb.Mean()
	return counts2class(avg, thresholds)
}

func (rb *RingBuffer) ModeLabel(thresholds []Bin) string {
	modeLab := OverflowLabel
	maxCount := 0
	labelMap := make(map[string]int)
	for _, val := range rb.data {
		class := counts2class(val, thresholds)
		if _, ok := labelMap[class]; !ok {
			labelMap[class] = 1
		} else {
			labelMap[class] += 1
		}
	}
	for label, count := range labelMap {
		if count > maxCount {
			modeLab = label
		}
	}
	return modeLab
}

func (rb *RingBuffer) String() string {
	return fmt.Sprintf("%.1f", rb.data)
}

func (cs *counter) run(ctx context.Context) error {
	freq := cs.frequency
	buffer := NewRingBuffer(cs.transitionCount)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			start := time.Now()
			// process for each stream in the list of cameras
			totalCounts := 0
			for camName, bbs := range cs.validRegions {
				// We ignore the metadata returned with the bytes
				imgBytes, _, err := cs.cams[camName].Image(ctx, utils.MimeTypePNG, nil)
				if err != nil {
					return errors.Errorf("camera %v error retrieving bytes: %q", camName, err)
				}
				img, err := png.Decode(bytes.NewReader(imgBytes))
				if err != nil {
					return errors.Errorf("camera %v error decoding bytes: %q", camName, err)
				}
				if len(bbs) == 0 { // if no bounding box, use the image without cropping
					dets, err := cs.detector.Detections(ctx, img, nil)
					if err != nil {
						return errors.Errorf("vision service error in background thread: %q", err)
					}
					c := cs.countDets(dets)
					totalCounts += c
				}
				for _, bb := range bbs {
					img = bb.Crop(img)
					dets, err := cs.detector.Detections(ctx, img, nil)
					if err != nil {
						return errors.Errorf("vision service error in background thread: %q", err)
					}
					c := cs.countDets(dets)
					totalCounts += c
				}
			}
			buffer.Add(float64(totalCounts))
			mean := buffer.Mean()
			meanClass := counts2class(mean, cs.thresholds)
			cs.syncVals.mu.Lock()
			cs.syncVals.numInView = totalCounts
			cs.syncVals.mean = mean
			cs.syncVals.class = meanClass
			cs.syncVals.countListString = buffer.String()
			cs.syncVals.mu.Unlock()
			took := time.Since(start)
			waitFor := time.Duration((1/freq)*float64(time.Second)) - took // only poll according to set freq
			if waitFor > time.Microsecond {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(waitFor):
				}
			}
		}
	}
}

// Readings contains both the label and the count of the underlying detector
func (cs *counter) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	select {
	case <-ctx.Done():
		return nil, errors.Wrap(ctx.Err(), "module might be configuring")
	case <-cs.cancelContext.Done():
		return nil, errors.Wrap(cs.cancelContext.Err(), "lost connection with background vision service loop")
	default:
		outMap := map[string]interface{}{}
		for k, v := range cs.extraFields {
			outMap[k] = v
		}
		cs.syncVals.mu.RLock()
		outMap["estimated_wait_time_min"] = cs.syncVals.class
		outMap["count_in_view"] = cs.syncVals.numInView
		outMap["mean_count"] = cs.syncVals.mean
		outMap["count_list"] = cs.syncVals.countListString
		cs.syncVals.mu.RUnlock()
		return outMap, nil
	}
}

// Close does nothing
func (cs *counter) Close(ctx context.Context) error {
	cs.cancelFunc()
	cs.activeBackgroundWorkers.Wait()
	return nil
}

// DoCommand implements nothing
func (cs *counter) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}
