package dns

import (
	"context"
	"fmt"
	"github.com/cloudflare/cloudflare-go"
	"strings"
)

func UpdateRecords(ctx context.Context, apiToken, fqdn, ipv4 string, port int) error {
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return err
	}

	zoneID, err := getZoneId(api, fqdn)
	if err != nil {
		return err
	}
	rc := cloudflare.ZoneIdentifier(zoneID)

	err = upsertIPv4Record(ctx, api, rc, fqdn, ipv4)
	if err != nil {
		return err
	}

	return upsertTXTRecord(ctx, api, rc, fqdn, port)
}

func getZoneId(api *cloudflare.API, fqdn string) (string, error) {
	labels := strings.Split(fqdn, ".")
	for i := range len(labels) - 1 {
		zoneName := strings.Join(labels[i:], ".")
		zoneID, err := api.ZoneIDByName(zoneName)
		if err == nil {
			return zoneID, nil
		}
	}
	return "", fmt.Errorf("zone not found for name: %s", fqdn)
}

func getRecordId(ctx context.Context, api *cloudflare.API, rc *cloudflare.ResourceContainer, typ, fqdn string) (string, error) {
	records, _, err := api.ListDNSRecords(ctx, rc, cloudflare.ListDNSRecordsParams{
		Name: fqdn,
		Type: typ,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list DNS records: %w", err)
	}
	if len(records) == 0 {
		return "", fmt.Errorf("no DNS record found for name=%s, type=%s", fqdn, typ)
	}
	if len(records) > 1 {
		return "", fmt.Errorf("multiple DNS records found for name: %s", fqdn)
	}
	record := records[0]

	return record.ID, nil
}

func upsertIPv4Record(ctx context.Context, api *cloudflare.API, rc *cloudflare.ResourceContainer, fqdn, newIPv4 string) error {
	aRecordID, err := getRecordId(ctx, api, rc, "A", fqdn)
	if err != nil {
		// Record doesn't exist, create it
		params := cloudflare.CreateDNSRecordParams{
			Type:    "A",
			Name:    fqdn,
			Content: newIPv4,
		}
		_, err := api.CreateDNSRecord(ctx, rc, params)
		if err != nil {
			return fmt.Errorf("failed to create A record: %w", err)
		}
		return nil
	}

	// Record exists, update it
	updateParams := cloudflare.UpdateDNSRecordParams{
		ID:      aRecordID,
		Type:    "A",
		Content: newIPv4,
	}

	_, err = api.UpdateDNSRecord(ctx, rc, updateParams)
	if err != nil {
		return fmt.Errorf("failed to update A record: %w", err)
	}

	return nil
}

func upsertTXTRecord(ctx context.Context, api *cloudflare.API, rc *cloudflare.ResourceContainer, fqdn string, port int) error {
	content := fmt.Sprintf("\"kcp-port=%d\"", port)
	txtRecordID, err := getRecordId(ctx, api, rc, "TXT", fqdn)
	if err != nil {
		// Record doesn't exist, create it
		params := cloudflare.CreateDNSRecordParams{
			Type:    "TXT",
			Name:    fqdn,
			Content: content,
		}
		_, err := api.CreateDNSRecord(ctx, rc, params)
		if err != nil {
			return fmt.Errorf("failed to create TXT record: %w", err)
		}
		return nil
	}

	// Record exists, update it
	updateParams := cloudflare.UpdateDNSRecordParams{
		ID:      txtRecordID,
		Type:    "TXT",
		Content: content,
	}

	_, err = api.UpdateDNSRecord(ctx, rc, updateParams)
	if err != nil {
		return fmt.Errorf("failed to update TXT record: %w", err)
	}

	return nil
}
