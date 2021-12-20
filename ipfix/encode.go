package ipfix

import (
	"bytes"
	"encoding/binary"
)

//Encode a Message to a IPFIX packet byte array.
func Encode(msg Message, seqNo uint32) []byte {

	if msg.Header.Length == 0 {
		fillHeaders(&msg)
	}

	buf := new(bytes.Buffer)
	//orginal flow header
	binary.Write(buf, binary.BigEndian, msg.Header.Version)
	binary.Write(buf, binary.BigEndian, msg.Header.Length)

	binary.Write(buf, binary.BigEndian, msg.Header.ExportTime)
	binary.Write(buf, binary.BigEndian, seqNo)
	binary.Write(buf, binary.BigEndian, msg.Header.DomainID)

	for _, template := range msg.TemplateSet {
		writeTemplateSet(buf, template)
	}
	for _, template := range msg.OptionsTemplateSet {
		writeOptionTemplateSet(buf, template)
	}
	writeDataSet(buf, msg.DataSet)

	result := buf.Bytes()
	return result
}

func writeTemplateSet(buf *bytes.Buffer, tplSet TemplateSet) {
	binary.Write(buf, binary.BigEndian, tplSet.Header.ID)
	binary.Write(buf, binary.BigEndian, tplSet.Header.Length)

	if len(tplSet.Templates) == 0 {
		return
	}
	for _, template := range tplSet.Templates {
		writeTemplate(buf, template)
	}
}

func writeTemplate(buf *bytes.Buffer, tplRecord TemplateRecord) {
	if tplRecord.FieldCount > 0 {
		binary.Write(buf, binary.BigEndian, tplRecord.ID)
		binary.Write(buf, binary.BigEndian, tplRecord.FieldCount)
		for _, field := range tplRecord.Fields {
			binary.Write(buf, binary.BigEndian, field.ID)
			binary.Write(buf, binary.BigEndian, field.Length)
			if field.ID >= 0x80 { // E == 1
				binary.Write(buf, binary.BigEndian, field.EnterpriseNo)
			}
		}
	}
}

func writeOptionTemplateSet(buf *bytes.Buffer, tplSet OptionsTemplateSet) {
	binary.Write(buf, binary.BigEndian, tplSet.Header.ID)
	binary.Write(buf, binary.BigEndian, tplSet.Header.Length)

	if len(tplSet.OptionTemplates) == 0 {
		return
	}
	for _, template := range tplSet.OptionTemplates {
		writeOptionsTemplate(buf, template)
	}
	for i := 0; i < tplSet.padding; i++ {
		binary.Write(buf, binary.BigEndian, PADDING)
	}
}

func writeOptionsTemplate(buf *bytes.Buffer, tplRecord OptionTemplateRecord) {
	if tplRecord.FieldCount > 0 {
		binary.Write(buf, binary.BigEndian, tplRecord.ID)
		binary.Write(buf, binary.BigEndian, tplRecord.FieldCount)
		binary.Write(buf, binary.BigEndian, tplRecord.ScopeFieldCount)
		for i := 0; i < int(tplRecord.ScopeFieldCount); i++ {
			binary.Write(buf, binary.BigEndian, (tplRecord.Fields[i]).ID)
			binary.Write(buf, binary.BigEndian, (tplRecord.Fields[i]).Length)
			if (tplRecord.Fields[i]).ID >= 0x80 { // E == 1
				binary.Write(buf, binary.BigEndian, (tplRecord.Fields[i]).EnterpriseNo)
			}
		}
		for i := int(tplRecord.ScopeFieldCount); i < int(tplRecord.FieldCount); i++ {
			binary.Write(buf, binary.BigEndian, (tplRecord.Fields[i]).ID)
			binary.Write(buf, binary.BigEndian, (tplRecord.Fields[i]).Length)
		}
	}
}

func writeDataSet(buf *bytes.Buffer, dataSet []DataSet) {
	for _, flowSet := range dataSet {
		binary.Write(buf, binary.BigEndian, flowSet.Header.ID)
		binary.Write(buf, binary.BigEndian, flowSet.Header.Length)
		for _, field := range flowSet.DataFields {
			binary.Write(buf, binary.BigEndian, field.Value)
		}
		for i := 0; i < flowSet.padding; i++ {
			binary.Write(buf, binary.BigEndian, PADDING)
		}
	}
}

//fill every head in message,including length,padding.
func fillHeaders(msg *Message) {
	length := uint16(16) //header 16 bytes

	//every sets length
	for i := range msg.TemplateSet {
		fillTemplate(&(msg.TemplateSet[i]))
		length += msg.TemplateSet[i].Header.Length
	}
	for i := range msg.OptionsTemplateSet {
		fillOptionTemplate(&(msg.OptionsTemplateSet[i]))
		length += msg.OptionsTemplateSet[i].Header.Length
	}
	for i := range msg.DataSet {
		fillDataSet(&(msg.DataSet[i]))
		length += msg.DataSet[i].Header.Length
	}

	msg.Header.Length = length
}

func fillTemplate(tplSet *TemplateSet) {
	length := uint16(4) //set head

	for _, tpl := range tplSet.Templates {
		length += 4 //t head
		for _, field := range tpl.Fields {
			length += 4 //field
			if field.ID > 0x80 {
				length += 4 //enterpriseNo
			}
		}
	}
	tplSet.Header.Length = length
}
func fillOptionTemplate(tplSet *OptionsTemplateSet) {
	length := uint16(4) //set head
	for _, tpl := range tplSet.OptionTemplates {
		if tpl.FieldCount > 0 {
			length += 6 //options template head
			for i := 0; i < int(tpl.ScopeFieldCount); i++ {
				length += 4 // id + length
				if tpl.Fields[i].ID > 0x80 {
					length += 4
				}
			}
			length += (tpl.FieldCount - tpl.ScopeFieldCount) * 4
		}
	}
	//padding
	if len(tplSet.OptionTemplates)%2 == 1 {
		length += 2
		tplSet.padding = 2
	}
	tplSet.Header.Length = length
}

//can not cal,val is interface{},need cal from template
//or cal by user
func fillDataSet(dataset *DataSet) {
	length := uint16(4) // set len
	for _, d := range dataset.DataFields {
		length += uint16(InfoModel[ElementKey{0, d.FieldID}].Type.minLen())
		//fmt.Printf("FieldID:%d,reflectLen:%d,InfoModelLen:%d\n", d.FieldID, reflect.TypeOf(d.Value).Size(), InfoModel[ElementKey{0, d.FieldID}].Type.minLen())
	}
	if length%4 != 0 {
		dataset.padding = int(4 - length%4)
		length += 4 - length%4
	}
	dataset.Header.Length = length
}
