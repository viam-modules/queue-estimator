package waitsensor

import (
	"context"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/vision"
	objdet "go.viam.com/rdk/vision/objectdetection"
	viamutils "go.viam.com/utils"
)

const (
	// ModelName is the name of the model
	ModelName = "wait-sensor"
	// OverflowLabel is the label if the counts exceed what was specified by the user
	OverflowLabel = "Overflow"
	// DefaulMaxFrequency is how often the vision service will poll the camera for a new image
	DefaultPollFrequency = 1.0
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

// Config contains names for necessary resources
type Config struct {
	DetectorName    string                 `json:"detector_name"`
	CameraName      string                 `json:"camera_name"`
	ChosenLabels    map[string]float64     `json:"chosen_labels"`
	CountThresholds map[string]int         `json:"count_thresholds"`
	PollFrequency   float64                `json:"poll_frequency_hz"`
	ExtraFields     map[string]interface{} `json:"extra_fields"`
}

// Validate validates the config and returns implicit dependencies,
// this Validate checks if the camera and detector exist for the module's vision model.
func (cfg *Config) Validate(path string) ([]string, error) {
	if cfg.DetectorName == "" {
		return nil, errors.New("attribute detector_name cannot be left blank")
	}
	if cfg.CameraName == "" {
		return nil, errors.New("attribute camera_name cannot be left blank")
	}
	if len(cfg.CountThresholds) == 0 {
		return nil, errors.New("attribute count_thresholds is required")
	}
	if cfg.PollFrequency < 0 {
		return nil, errors.New("attribute poll_frequency_hz cannot be negative")
	}
	testMap := map[int]string{}
	for label, v := range cfg.CountThresholds {
		if _, ok := testMap[v]; ok {
			return nil, errors.Errorf("cannot have two labels for the same threshold in count_thresholds. Threshold value %v appears more than once", v)
		}
		testMap[v] = label
	}
	return []string{cfg.DetectorName, cfg.CameraName}, nil
}

// Bin stores the thresholds that turns counts into labels
type Bin struct {
	UpperBound int
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
		b := Bin{key, thresholds[key]}
		out = append(out, b)
	}
	return out
}

type counter struct {
	resource.Named
	cancelFunc              context.CancelFunc
	cancelContext           context.Context
	activeBackgroundWorkers sync.WaitGroup
	logger                  logging.Logger
	detName                 string
	camName                 string
	detector                vision.Service
	labels                  map[string]float64
	thresholds              []Bin
	frequency               float64
	num                     atomic.Int64
	class                   atomic.Value
	extraFields             map[string]interface{}
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

	countConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return errors.Errorf("Could not assert proper config for %s", ModelName)
	}
	cs.frequency = DefaultPollFrequency
	if countConf.PollFrequency > 0 {
		cs.frequency = countConf.PollFrequency
	}
	cs.extraFields = map[string]interface{}{}
	if countConf.ExtraFields != nil {
		cs.extraFields = countConf.ExtraFields
	}
	cs.camName = countConf.CameraName
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

func (cs *counter) count(dets []objdet.Detection) (string, int) {
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
	// associated the number with the right label
	for _, thresh := range cs.thresholds {
		if count <= thresh.UpperBound {
			return thresh.Label, count
		}
	}
	return OverflowLabel, count
}

func (cs *counter) run(ctx context.Context) error {
	freq := cs.frequency
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			start := time.Now()
			dets, err := cs.detector.DetectionsFromCamera(ctx, cs.camName, nil)
			if err != nil {
				return errors.Errorf("vision service error in background thread: %q", err)
			}
			class, num := cs.count(dets)
			cs.class.Store(class)
			cs.num.Store(int64(num))
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
		className, ok := cs.class.Load().(string)
		if !ok {
			return nil, errors.Errorf("class string was not a string, but %T", className)
		}
		countNumber := cs.num.Load()
		outMap["estimated_wait_time_min"] = className
		outMap["people_in_line"] = countNumber
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
