# cups_printer

Creates a persistent driverless queue and manages its device URI, description, and location. The CUPS server must be able to reach the IPP device while generating its driverless PPD. Queue creation waits for a PPD and reapplies metadata afterward so CUPS does not replace an empty description with a model name.

```hcl
resource "cups_printer" "office" {
  name        = "poc-office"
  device_uri  = "ipp://printer.local:8000/ipp/print"
  description = "Office printer"
  location    = "First floor"
}
```

| Attribute | Type | Behavior |
| --- | --- | --- |
| `name` | Required string | 1–127 ASCII letters, digits, underscores or hyphens; begins with a letter/digit. Change requires replacement. |
| `device_uri` | Required string | IPP/IPPS URI with hostname, at most 1023 bytes, no credentials/query/fragment. Change requires replacement. |
| `description` | Optional string | Default empty. Omission/null/empty clears it. Valid UTF-8, at most 127 bytes, no NUL characters. Updates in place. |
| `location` | Optional string | Same clearing and validation rules as description. Updates in place. |
| `id` | Computed string | Queue name within the selected server. |

Creation selects CUPS' `everywhere` model. Legacy PPD uploads, raw queues, non-IPP device backends, queue sharing, job defaults, enablement, and accepting-jobs settings are not managed. New queues are paused and reject jobs in the tested CUPS configuration. No test pages are submitted.

Existing queue names cause a create error directing the user to import. There is an unavoidable race with external creators because CUPS' add/modify operation is not atomic create-only. Avoid concurrent management of the same queue across Terraform states or administrative tools. A lost mutation response is reported without replaying the mutation; inspect the queue and import it if creation succeeded remotely.

A definitively missing queue is removed from state on refresh. Authentication, network, and other errors fail the refresh and preserve state. Description/location drift is corrected in place; name/device changes require replacement. If creation was acknowledged but PPD verification fails, the provider returns partial state so the queue remains tracked for cleanup.

## Import

Use the existing queue's name under the configured provider endpoint:

```sh
terraform import cups_printer.office poc-office
```

The read path recovers name, device URI, description, and location. Imported queues must have a supported, credential-free IPP/IPPS device URI. The provider does not recover or validate an imported queue's driver/model, and metadata updates preserve that existing model. Replacements create a new driverless queue.

## Delete

Destroy removes the managed queue; an already absent queue is treated as success. Replacement normally destroys the old queue before creating the new one. Queue deletion can discard pending jobs under CUPS behavior, so do not use the POC on active production queues. The test loop exercises only empty queues and does not verify job handling.
