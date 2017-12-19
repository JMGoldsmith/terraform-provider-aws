package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/gamelift"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsGameliftFleet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGameliftFleetCreate,
		Read:   resourceAwsGameliftFleetRead,
		Update: resourceAwsGameliftFleetUpdate,
		Delete: resourceAwsGameliftFleetDelete,

		// TODO: Timeout

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"build_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"ec2_instance_type": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ec2_inbound_permission": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_port": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"ip_range": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"protocol": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"to_port": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
			"log_paths": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"metric_groups": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"new_game_session_protection_policy": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  gamelift.ProtectionPolicyNoProtection,
			},
			"operating_system": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"peer_vpc_aws_account_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"peer_vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"resource_creation_limit_policy": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"new_game_sessions_per_creator": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"policy_period_in_minutes": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
			"runtime_configuration": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"game_session_activation_timeout_seconds": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"max_concurrent_game_session_activations": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"server_process": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"concurrent_executions": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"launch_path": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"parameters": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},

			// TODO: Deprecated?
			"server_launch_parameters": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"server_launch_path": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceAwsGameliftFleetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	input := gamelift.CreateFleetInput{
		BuildId:         aws.String(d.Get("build_id").(string)),
		EC2InstanceType: aws.String(d.Get("ec2_instance_type").(string)),
		Name:            aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("ec2_inbound_permission"); ok {
		input.EC2InboundPermissions = expandGameliftIpPermissions(v.([]interface{}))
	}
	if v, ok := d.GetOk("log_paths"); ok {
		input.LogPaths = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("metric_groups"); ok {
		input.MetricGroups = expandStringList(v.([]interface{}))
	}
	if v, ok := d.GetOk("new_game_session_protection_policy"); ok {
		input.NewGameSessionProtectionPolicy = aws.String(v.(string))
	}
	if v, ok := d.GetOk("peer_vpc_aws_account_id"); ok {
		input.PeerVpcAwsAccountId = aws.String(v.(string))
	}
	if v, ok := d.GetOk("peer_vpc_id"); ok {
		input.PeerVpcId = aws.String(v.(string))
	}
	if v, ok := d.GetOk("resource_creation_limit_policy"); ok {
		input.ResourceCreationLimitPolicy = expandGameliftResourceCreationLimitPolicy(v.([]interface{}))
	}
	if v, ok := d.GetOk("runtime_configuration"); ok {
		input.RuntimeConfiguration = expandGameliftRuntimeConfiguration(v.([]interface{}))
	}
	if v, ok := d.GetOk("server_launch_parameters"); ok {
		input.ServerLaunchParameters = aws.String(v.(string))
	}
	if v, ok := d.GetOk("server_launch_path"); ok {
		input.ServerLaunchPath = aws.String(v.(string))
	}

	log.Printf("[INFO] Creating Gamelift Fleet: %s", input)
	out, err := conn.CreateFleet(&input)
	if err != nil {
		return err
	}

	d.SetId(*out.FleetAttributes.FleetId)

	stateConf := &resource.StateChangeConf{
		Pending: []string{
			gamelift.FleetStatusActivating,
			gamelift.FleetStatusBuilding,
			gamelift.FleetStatusDownloading,
			gamelift.FleetStatusNew,
			gamelift.FleetStatusValidating,
		},
		Target:  []string{gamelift.FleetStatusActive},
		Timeout: 15 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeFleetAttributes(&gamelift.DescribeFleetAttributesInput{
				FleetIds: aws.StringSlice([]string{d.Id()}),
			})
			if err != nil {
				return 42, "", err
			}

			attributes := out.FleetAttributes
			if len(attributes) < 1 {
				return nil, "", nil
			}
			if len(attributes) != 1 {
				return 42, "", fmt.Errorf("Expected exactly 1 Gamelift fleet, found %d under %q",
					len(attributes), d.Id())
			}

			fleet := attributes[0]
			return fleet, *fleet.Status, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		events, err := getGameliftFleetFailures(conn, d.Id())
		if err != nil {
			log.Printf("[WARN] Failed to poll fleet failures: %s", err)
		}
		if len(events) > 0 {
			return fmt.Errorf("%s Recent events: %q", err, events)
		}

		return err
	}

	return resourceAwsGameliftFleetRead(d, meta)
}

func resourceAwsGameliftFleetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	log.Printf("[INFO] Describing Gamelift Fleet: %s", d.Id())
	out, err := conn.DescribeFleetAttributes(&gamelift.DescribeFleetAttributesInput{
		FleetIds: aws.StringSlice([]string{d.Id()}),
	})
	if err != nil {
		return err
	}
	attributes := out.FleetAttributes
	if len(attributes) < 1 {
		log.Printf("[WARN] Gamelift Fleet (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if len(attributes) != 1 {
		return fmt.Errorf("Expected exactly 1 Gamelift fleet, found %d under %q",
			len(attributes), d.Id())
	}
	fleet := attributes[0]

	d.Set("build_id", fleet.BuildId)
	d.Set("description", fleet.Description)
	d.Set("fleet_arn", fleet.FleetArn)
	d.Set("log_paths", flattenStringList(fleet.LogPaths))
	d.Set("metric_groups", flattenStringList(fleet.MetricGroups))
	d.Set("name", fleet.Name)
	d.Set("new_game_session_protection_policy", fleet.NewGameSessionProtectionPolicy)
	d.Set("operating_system", fleet.OperatingSystem)
	d.Set("resource_creation_limit_policy", flattenGameliftResourceCreationLimitPolicy(fleet.ResourceCreationLimitPolicy))
	d.Set("server_launch_parameters", fleet.ServerLaunchParameters)
	d.Set("server_launch_path", fleet.ServerLaunchPath)

	return nil
}

func resourceAwsGameliftFleetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	log.Printf("[INFO] Updating Gamelift Fleet: %s", d.Id())
	_, err := conn.UpdateFleetAttributes(&gamelift.UpdateFleetAttributesInput{
		Description:  aws.String(d.Get("description").(string)),
		FleetId:      aws.String(d.Get("fleet_id").(string)),
		MetricGroups: expandStringList(d.Get("metric_groups").([]interface{})),
		Name:         aws.String(d.Get("name").(string)),
		NewGameSessionProtectionPolicy: aws.String(d.Get("new_game_session_protection_policy").(string)),
		ResourceCreationLimitPolicy:    expandGameliftResourceCreationLimitPolicy(d.Get("resource_creation_limit_policy").([]interface{})),
	})
	if err != nil {
		return err
	}

	return resourceAwsGameliftFleetRead(d, meta)
}

func resourceAwsGameliftFleetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).gameliftconn

	log.Printf("[INFO] Deleting Gamelift Fleet: %s", d.Id())
	_, err := conn.DeleteFleet(&gamelift.DeleteFleetInput{
		FleetId: aws.String(d.Id()),
	})
	if err != nil {
		return err
	}

	stateConf := resource.StateChangeConf{
		Pending: []string{
			gamelift.FleetStatusActive,
			gamelift.FleetStatusDeleting,
			gamelift.FleetStatusError,
			gamelift.FleetStatusTerminated,
		},
		Target:  []string{},
		Timeout: 15 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeFleetAttributes(&gamelift.DescribeFleetAttributesInput{
				FleetIds: aws.StringSlice([]string{d.Id()}),
			})
			if err != nil {
				return 42, "", err
			}

			attributes := out.FleetAttributes
			if len(attributes) < 1 {
				return nil, "", nil
			}
			if len(attributes) != 1 {
				return 42, "", fmt.Errorf("Expected exactly 1 Gamelift fleet, found %d under %q",
					len(attributes), d.Id())
			}

			fleet := attributes[0]
			return fleet, *fleet.Status, nil
		},
	}
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return nil
}

func expandGameliftIpPermissions(cfgs []interface{}) []*gamelift.IpPermission {
	if len(cfgs) < 1 {
		return []*gamelift.IpPermission{}
	}

	perms := make([]*gamelift.IpPermission, len(cfgs), len(cfgs))
	for i, rawCfg := range cfgs {
		cfg := rawCfg.(map[string]interface{})
		perms[i] = &gamelift.IpPermission{
			FromPort: aws.Int64(int64(cfg["from_port"].(int))),
			IpRange:  aws.String(cfg["ip_range"].(string)),
			Protocol: aws.String(cfg["protocol"].(string)),
			ToPort:   aws.Int64(int64(cfg["to_port"].(int))),
		}
	}
	return perms
}

func flattenGameliftIpPermissions(ipps []*gamelift.IpPermission) []interface{} {
	perms := make([]interface{}, len(ipps), len(ipps))

	for i, ipp := range ipps {
		m := make(map[string]interface{}, 0)
		m["from_port"] = *ipp.FromPort
		m["ip_range"] = *ipp.IpRange
		m["protocol"] = *ipp.Protocol
		m["to_port"] = *ipp.ToPort
		perms[i] = m
	}

	return perms
}

func expandGameliftResourceCreationLimitPolicy(cfg []interface{}) *gamelift.ResourceCreationLimitPolicy {
	if len(cfg) < 1 {
		return nil
	}
	out := gamelift.ResourceCreationLimitPolicy{}
	m := cfg[0].(map[string]interface{})

	if v, ok := m["new_game_sessions_per_creator"]; ok {
		out.NewGameSessionsPerCreator = aws.Int64(int64(v.(int)))
	}
	if v, ok := m["policy_period_in_minutes"]; ok {
		out.PolicyPeriodInMinutes = aws.Int64(int64(v.(int)))
	}

	return &out
}

func flattenGameliftResourceCreationLimitPolicy(policy *gamelift.ResourceCreationLimitPolicy) []interface{} {
	if policy == nil {
		return []interface{}{}
	}

	m := make(map[string]interface{}, 0)
	m["new_game_sessions_per_creator"] = *policy.NewGameSessionsPerCreator
	m["policy_period_in_minutes"] = *policy.PolicyPeriodInMinutes

	return []interface{}{m}
}

func expandGameliftRuntimeConfiguration(cfg []interface{}) *gamelift.RuntimeConfiguration {
	if len(cfg) < 1 {
		return nil
	}
	out := gamelift.RuntimeConfiguration{}
	m := cfg[0].(map[string]interface{})

	if v, ok := m["game_session_activation_timeout_seconds"]; ok {
		out.GameSessionActivationTimeoutSeconds = aws.Int64(int64(v.(int)))
	}
	if v, ok := m["max_concurrent_game_session_activations"]; ok {
		out.MaxConcurrentGameSessionActivations = aws.Int64(int64(v.(int)))
	}
	if v, ok := m["server_process"]; ok {
		out.ServerProcesses = expandGameliftServerProcesses(v.([]interface{}))
	}

	return &out
}

func expandGameliftServerProcesses(cfgs []interface{}) []*gamelift.ServerProcess {
	if len(cfgs) < 1 {
		return []*gamelift.ServerProcess{}
	}

	processes := make([]*gamelift.ServerProcess, len(cfgs), len(cfgs))
	for i, rawCfg := range cfgs {
		cfg := rawCfg.(map[string]interface{})
		process := &gamelift.ServerProcess{
			ConcurrentExecutions: aws.Int64(int64(cfg["concurrent_executions"].(int))),
			LaunchPath:           aws.String(cfg["launch_path"].(string)),
		}
		if v, ok := cfg["parameters"]; ok {
			process.Parameters = aws.String(v.(string))
		}
		processes[i] = process
	}
	return processes
}

func getGameliftFleetFailures(conn *gamelift.GameLift, id string) ([]*gamelift.Event, error) {
	var events []*gamelift.Event
	err := _getGameliftFleetFailures(conn, id, nil, events)
	return events, err
}

func _getGameliftFleetFailures(conn *gamelift.GameLift, id string, nextToken *string, events []*gamelift.Event) error {
	eOut, err := conn.DescribeFleetEvents(&gamelift.DescribeFleetEventsInput{
		FleetId:   aws.String(id),
		NextToken: nextToken,
	})
	if err != nil {
		return err
	}

	for _, e := range eOut.Events {
		if isGameliftEventFailure(e) {
			events = append(events, e)
		}
	}

	if eOut.NextToken != nil {
		err := _getGameliftFleetFailures(conn, id, nextToken, events)
		if err != nil {
			return err
		}
	}

	return nil
}

func isGameliftEventFailure(event *gamelift.Event) bool {
	failureCodes := []string{
		"FLEET_STATE_ERROR",
		"FLEET_INITIALIZATION_FAILED",
		"FLEET_BINARY_DOWNLOAD_FAILED",
		"FLEET_VALIDATION_LAUNCH_PATH_NOT_FOUND",
		"FLEET_VALIDATION_EXECUTABLE_RUNTIME_FAILURE",
		"FLEET_VALIDATION_TIMED_OUT",
		"FLEET_ACTIVATION_FAILED",
		"FLEET_ACTIVATION_FAILED_NO_INSTANCES",
		"SERVER_PROCESS_INVALID_PATH",
		"SERVER_PROCESS_SDK_INITIALIZATION_TIMEOUT",
		"SERVER_PROCESS_PROCESS_READY_TIMEOUT",
		"SERVER_PROCESS_CRASHED",
		"SERVER_PROCESS_TERMINATED_UNHEALTHY",
		"SERVER_PROCESS_FORCE_TERMINATED",
		"SERVER_PROCESS_PROCESS_EXIT_TIMEOUT",
		"GAME_SESSION_ACTIVATION_TIMEOUT",
		"FLEET_VPC_PEERING_FAILED",
	}
	for _, fc := range failureCodes {
		if *event.EventCode == fc {
			return true
		}
	}
	return false
}
