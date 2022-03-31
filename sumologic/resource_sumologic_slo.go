package sumologic

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"log"
	"regexp"
)

const sloAggregationRegexString = `^(Avg|Min|Max|Sum|(p[5-9][0-9])(\.\d{1,3})?$)$`
const sloAggregationWindowRegexString = `^[0-9]{1,2}(m|h)$` // TODO make it exact of min 1m and max 1h
const sloContentType = "slo"

func resourceSumologicSLO() *schema.Resource {

	aggrRegex := regexp.MustCompile(sloAggregationRegexString)
	windowRegex := regexp.MustCompile(sloAggregationWindowRegexString)

	queryGroupElemSchema := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"row_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"query": {
				Type:     schema.TypeString,
				Required: true,
			},
			"use_row_count": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"field": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}

	return &schema.Resource{
		Create: resourceSumologicSLOCreate,
		Read:   resourceSumologicSLORead,
		Update: resourceSumologicSLOUpdate,
		Delete: resourceSumologicSLODelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Required: true,
			},
			"version": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"created_by": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"modified_at": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"modified_by": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"parent_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"is_system": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			//"content_type": {
			//	Type:     schema.TypeString,
			//	Optional: true,
			//	Default:  "Slo",
			//	ExactlyOneOf: []string{
			//		"Slo",
			//	},
			//},
			//"type": {
			//	Type:     schema.TypeString,
			//	Optional: true,
			//	Default:  "SlosLibraryFolder",
			//	ExactlyOneOf: []string{
			//		"SlosLibrarySlo",
			//		"SlosLibrarySloUpdate",
			//	},
			//},
			"signal_type": {
				Type:     schema.TypeString,
				Optional: true,
				ExactlyOneOf: []string{
					"Latency", "Error", "Throughput", "Availability", "Other",
				},
			},
			"compliance": {
				Type:     schema.TypeMap,
				Optional: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Optional: false,
							ExactlyOneOf: []string{
								"Rolling",
								"Calendar",
							},
						},
						"target": {
							Type:         schema.TypeInt,
							Optional:     false,
							ValidateFunc: validation.IntBetween(0, 100),
						},
						"timezone": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"size": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: validation.StringInSlice([]string{
								"1d", "2d", "3d", "4d", "5d", "6d", "7d", "8d", "9d", "10d", "11d", "12d", "13d", "14d",
							}, false),
						},
					},
				},
			},
			"indicator": {
				Type:     schema.TypeMap,
				Optional: false,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"evaluation_type": {
							Type:     schema.TypeString,
							Optional: false,
							ExactlyOneOf: []string{
								"Threshold",
								"Range",
							},
						},
						"queries": {
							Type:     schema.TypeList,
							Required: true,
							MinItems: 1,
							MaxItems: 2,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"query_group_type": {
										Type:     schema.TypeString,
										Required: true,
										ValidateFunc: validation.StringInSlice([]string{
											"Successful", "Unsuccessful", "Total", "Threshold",
										}, false),
									},
									"query_group": {
										Type:     schema.TypeList,
										Required: true,
										Elem:     queryGroupElemSchema,
									},
								},
							},
						},
						"queryType": {
							Type:     schema.TypeInt,
							Optional: false,
							ValidateFunc: validation.StringInSlice([]string{
								"Logs", "Metrics",
							}, false),
						},
						"threshold": {
							Type:     schema.TypeFloat,
							Optional: false,
						},
						"op": {
							Type:     schema.TypeInt,
							Optional: false,
							ValidateFunc: validation.StringInSlice([]string{
								"LessThan", "GreaterThan", "LessThanOrEqual", "GreaterThanOrEqual",
							}, false),
						},
						"aggregation": {
							Type:         schema.TypeInt,
							Optional:     false,
							ValidateFunc: validation.StringMatch(aggrRegex, `value must match : `+sloAggregationRegexString),
						},
						"size": {
							Type:         schema.TypeInt,
							Optional:     false,
							ValidateFunc: validation.StringMatch(windowRegex, `value must match : `+sloAggregationRegexString),
						},
					},
				},
			},
			"is_mutable": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"is_locked": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"service": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"application": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"post_request_map": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceSumologicSLOCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	if d.Id() == "" {
		slo := resourceToSLO(d)
		slo.Type = "SlosLibrarySlo"
		if slo.ParentID == "" {
			rootFolder, err := c.GetSLOLibraryFolder("root")
			if err != nil {
				return err
			}

			slo.ParentID = rootFolder.ID
		}
		paramMap := map[string]string{
			"parentId": slo.ParentID,
		}
		sloDefinitionID, err := c.CreateSLO(slo, paramMap)
		if err != nil {
			return err
		}

		d.SetId(sloDefinitionID)
	}
	return resourceSumologicSLORead(d, meta)
}

func resourceSumologicSLORead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)

	slo, err := c.SLORead(d.Id(), nil)
	if err != nil {
		return err
	}

	if slo == nil {
		log.Printf("[WARN] SLO not found, removing from state: %v - %v", d.Id(), err)
		d.SetId("")
		return nil
	}

	d.Set("name", slo.Name)
	d.Set("description", slo.Description)
	d.Set("version", slo.Version)
	d.Set("created_at", slo.CreatedAt)
	d.Set("created_by", slo.CreatedBy)
	d.Set("modified_at", slo.ModifiedAt)
	d.Set("modified_by", slo.ModifiedBy)
	d.Set("parent_id", slo.ParentID)
	d.Set("content_type", "slo")
	d.Set("is_mutable", slo.IsMutable)
	d.Set("is_locked", slo.IsLocked)
	d.Set("is_system", slo.IsSystem)
	d.Set("service", slo.Service)
	d.Set("application", slo.Application)
	// set compliance
	if err := d.Set("compliance", slo.Compliance); err != nil {
		return fmt.Errorf("error setting fields for resource %s: %s", d.Id(), err)
	}
	if err := d.Set("indicator", slo.Indicator); err != nil {
		return fmt.Errorf("error setting fields for resource %s: %s", d.Id(), err)
	}

	return nil
}

func resourceToSLO(d *schema.ResourceData) SLOLibrarySLO {
	compliance := getSLOCompliance(d)
	indicator := getSLOIndicator(d)
	return SLOLibrarySLO{
		ID:          d.Id(),
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Version:     d.Get("version").(int),
		CreatedAt:   d.Get("created_at").(string),
		CreatedBy:   d.Get("created_by").(string),
		ModifiedAt:  d.Get("modified_at").(string),
		ModifiedBy:  d.Get("modified_by").(string),
		ParentID:    d.Get("parent_id").(string),
		ContentType: d.Get("content_type").(string),
		Type:        d.Get("type").(string),
		IsSystem:    d.Get("is_system").(bool),
		IsMutable:   d.Get("is_mutable").(bool),
		IsLocked:    d.Get("is_locked").(bool),
		SignalType:  d.Get("signal_type").(string),
		Compliance:  compliance,
		Indicator:   indicator,
		Service:     d.Get("service").(string),
		Application: d.Get("application").(string),
	}
}

func getSLOCompliance(d *schema.ResourceData) SLOCompliance {
	complianceDict := d.Get("compliance").(map[string]interface{})
	return SLOCompliance{
		ComplianceType: complianceDict["compliance_type"].(string),
		Target:         complianceDict["target"].(int),
		Timezone:       complianceDict["timezone"].(string),
		Size:           complianceDict["size"].(string),
	}
}

func getSLOIndicator(d *schema.ResourceData) SLOIndicator {
	indicatorDict := d.Get("indicator").(map[string]interface{})
	return SLOIndicator{
		EvaluationType: indicatorDict["evaluation_type"].(string),
		QueryType:      indicatorDict["query_type"].(string),
		Queries:        GetSLOIndicatorQueries(d),
	}
}

func GetSLOIndicatorQueries(d *schema.ResourceData) []SLIQueryGroup {

	queriesRaw := d.Get("queries").([]interface{})
	queries := make([]SLIQueryGroup, len(queriesRaw))

	for i := range queries {
		qDict := queriesRaw[i].(map[string]interface{})

		queries[i].QueryGroupType = qDict["query_group_type"].(string)

		qGroupRaw := qDict["query_group"].([]interface{})
		qGroups := make([]SLIQuery, len(qGroupRaw))

		for j := range qGroups {
			qGroup := qGroupRaw[i].(SLIQuery)
			qGroups[j] = qGroup
		}
		queries[i].QueryGroup = qGroups
	}

	return queries
}

func resourceSumologicSLOUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	slo := resourceToSLO(d)
	slo.Type = "SlosLibrarySloUpdate"
	err := c.UpdateSLO(slo)
	if err != nil {
		return err
	}
	return resourceSumologicSLORead(d, meta)
}

func resourceSumologicSLODelete(d *schema.ResourceData, meta interface{}) error {

	c := meta.(*Client)
	slo := resourceToSLO(d)
	err := c.DeleteSLO(slo.ID)
	if err != nil {
		return err
	}
	return nil
}